package cart

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/gorm"
)

const cartStatusActive = "ACTIVE"

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type AddItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type UpdateItemRequest struct {
	Quantity int `json:"quantity"`
}

func (s *Service) getOrCreateActiveCart(customerID uuid.UUID) (*database.Cart, error) {
	var c database.Cart
	if err := s.db.Where("customer_id = ? and status = ?", customerID, cartStatusActive).First(&c).Error; err == nil {
		return &c, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	c = database.Cart{CustomerID: customerID, Status: cartStatusActive}
	if err := s.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) AddItem(customerID uuid.UUID, req AddItemRequest) error {
	if req.Quantity <= 0 {
		return errors.New("VALIDATION:quantity must be > 0")
	}
	pid, err := uuid.Parse(req.ProductID)
	if err != nil {
		return errors.New("VALIDATION:product_id must be valid uuid")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var p database.Product
		if err := tx.Where("id = ?", pid).First(&p).Error; err != nil {
			return errors.New("NOT_FOUND:product")
		}
		if !p.IsActive {
			return errors.New("VALIDATION:product is inactive")
		}
		if req.Quantity > p.Stock {
			return errors.New("VALIDATION:quantity exceeds stock")
		}
		cart, err := s.getOrCreateActiveCart(customerID)
		if err != nil {
			return err
		}
		var item database.CartItem
		err = tx.Where("cart_id = ? and product_id = ?", cart.ID, pid).First(&item).Error
		if err == nil {
			newQty := item.Quantity + req.Quantity
			if newQty > p.Stock {
				return errors.New("VALIDATION:quantity exceeds stock")
			}
			item.Quantity = newQty
			item.PriceSnapshotAmount = p.PriceAmount
			return tx.Save(&item).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&database.CartItem{CartID: cart.ID, ProductID: pid, Quantity: req.Quantity, PriceSnapshotAmount: p.PriceAmount}).Error
	})
}

func (s *Service) UpdateItem(customerID, itemID uuid.UUID, qty int) error {
	if qty <= 0 {
		return errors.New("VALIDATION:quantity must be > 0")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var cart database.Cart
		if err := tx.Where("customer_id = ? and status = ?", customerID, cartStatusActive).First(&cart).Error; err != nil {
			return errors.New("NOT_FOUND:cart")
		}
		var item database.CartItem
		if err := tx.Where("id = ? and cart_id = ?", itemID, cart.ID).First(&item).Error; err != nil {
			return errors.New("NOT_FOUND:item")
		}
		var p database.Product
		if err := tx.Where("id = ?", item.ProductID).First(&p).Error; err != nil {
			return errors.New("NOT_FOUND:product")
		}
		if !p.IsActive {
			return errors.New("VALIDATION:product is inactive")
		}
		if qty > p.Stock {
			return errors.New("VALIDATION:quantity exceeds stock")
		}
		item.Quantity = qty
		item.PriceSnapshotAmount = p.PriceAmount
		return tx.Save(&item).Error
	})
}

func (s *Service) RemoveItem(customerID, itemID uuid.UUID) error {
	var cart database.Cart
	if err := s.db.Where("customer_id = ? and status = ?", customerID, cartStatusActive).First(&cart).Error; err != nil {
		return errors.New("NOT_FOUND:cart")
	}
	res := s.db.Where("id = ? and cart_id = ?", itemID, cart.ID).Delete(&database.CartItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND:item")
	}
	return nil
}

func (s *Service) Clear(customerID uuid.UUID) error {
	var cart database.Cart
	if err := s.db.Where("customer_id = ? and status = ?", customerID, cartStatusActive).First(&cart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return s.db.Where("cart_id = ?", cart.ID).Delete(&database.CartItem{}).Error
}

func (s *Service) Get(customerID uuid.UUID) (map[string]interface{}, error) {
	cart, err := s.getOrCreateActiveCart(customerID)
	if err != nil {
		return nil, err
	}
	var items []database.CartItem
	if err := s.db.Where("cart_id = ?", cart.ID).Find(&items).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ProductID)
	}
	var products []database.Product
	if len(ids) > 0 {
		_ = s.db.Where("id in ?", ids).Find(&products).Error
	}
	pmap := map[uuid.UUID]database.Product{}
	for _, p := range products {
		pmap[p.ID] = p
	}
	respItems := make([]map[string]interface{}, 0, len(items))
	var total int64
	for _, it := range items {
		sub := int64(it.Quantity) * it.PriceSnapshotAmount
		total += sub
		respItems = append(respItems, map[string]interface{}{"id": it.ID, "product_id": it.ProductID, "quantity": it.Quantity, "price_snapshot_amount": it.PriceSnapshotAmount, "subtotal_amount": sub, "product": pmap[it.ProductID]})
	}
	return map[string]interface{}{"id": cart.ID, "customer_id": cart.CustomerID, "status": cart.Status, "items": respItems, "subtotal_amount": total, "total_amount": total}, nil
}

func HandleErr(err error) (int, string, string, []string) {
	m := err.Error()
	switch {
	case strings.HasPrefix(m, "VALIDATION:"):
		return 400, "Validation error", "VALIDATION_ERROR", []string{strings.TrimPrefix(m, "VALIDATION:")}
	case strings.HasPrefix(m, "NOT_FOUND:"):
		return 404, "Not found", "NOT_FOUND", []string{strings.TrimPrefix(m, "NOT_FOUND:") + " not found"}
	default:
		return 500, "Internal server error", "INTERNAL_ERROR", nil
	}
}
