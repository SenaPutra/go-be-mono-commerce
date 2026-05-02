package order

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusPendingPayment = "PENDING_PAYMENT"
	StatusPaid           = "PAID"
	StatusProcessing     = "PROCESSING"
	StatusReadyToShip    = "READY_TO_SHIP"
	StatusShipped        = "SHIPPED"
	StatusCompleted      = "COMPLETED"
	StatusCancelled      = "CANCELLED"
	StatusExpired        = "EXPIRED"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CheckoutRequest struct {
	AddressID string `json:"address_id"`
	Notes     string `json:"notes"`
}

func (s *Service) Checkout(customerID uuid.UUID, req CheckoutRequest) (*database.Order, error) {
	addrID, err := uuid.Parse(req.AddressID)
	if err != nil {
		return nil, errors.New("VALIDATION:address_id must be valid uuid")
	}
	var out database.Order
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var addr database.CustomerAddress
		if err := tx.Where("id = ? and customer_id = ?", addrID, customerID).First(&addr).Error; err != nil {
			return errors.New("VALIDATION:address does not belong to customer")
		}
		var cart database.Cart
		if err := tx.Where("customer_id = ? and status = ?", customerID, "ACTIVE").First(&cart).Error; err != nil {
			return errors.New("NOT_FOUND:active cart")
		}
		var items []database.CartItem
		if err := tx.Where("cart_id = ?", cart.ID).Find(&items).Error; err != nil {
			return err
		}
		if len(items) == 0 {
			return errors.New("VALIDATION:cart is empty")
		}
		productIDs := make([]uuid.UUID, 0, len(items))
		for _, it := range items {
			productIDs = append(productIDs, it.ProductID)
		}
		var products []database.Product
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id in ?", productIDs).Find(&products).Error; err != nil {
			return err
		}
		pmap := map[uuid.UUID]database.Product{}
		for _, p := range products {
			pmap[p.ID] = p
		}
		var total int64
		for _, it := range items {
			p, ok := pmap[it.ProductID]
			if !ok {
				return errors.New("NOT_FOUND:product")
			}
			if !p.IsActive {
				return fmt.Errorf("VALIDATION:product %s is inactive", p.ID.String())
			}
			if p.Stock < it.Quantity {
				return fmt.Errorf("VALIDATION:insufficient stock for product %s", p.ID.String())
			}
			sub := int64(it.Quantity) * it.PriceSnapshotAmount
			total += sub
		}
		order := database.Order{CustomerID: customerID, OrderNumber: generateOrderNumber(), TotalAmount: total, Status: StatusPendingPayment}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for _, it := range items {
			p := pmap[it.ProductID]
			sub := int64(it.Quantity) * it.PriceSnapshotAmount
			if err := tx.Create(&database.OrderItem{OrderID: order.ID, ProductID: p.ID, ProductNameSnapshot: p.Name, Quantity: it.Quantity, PriceAmount: it.PriceSnapshotAmount, SubtotalAmount: sub}).Error; err != nil {
				return err
			}
			if err := tx.Model(&database.Product{}).Where("id = ?", p.ID).Update("stock", gorm.Expr("stock - ?", it.Quantity)).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&database.Cart{}).Where("id = ?", cart.ID).Update("status", "CHECKED_OUT").Error; err != nil {
			return err
		}
		out = order
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func generateOrderNumber() string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("ORD-%s-%s", time.Now().UTC().Format("20060102"), string(b))
}

func (s *Service) ListCustomerOrders(customerID uuid.UUID) ([]database.Order, error) {
	var orders []database.Order
	return orders, s.db.Where("customer_id = ?", customerID).Order("created_at desc").Find(&orders).Error
}

func (s *Service) GetCustomerOrder(customerID, orderID uuid.UUID) (*database.Order, error) {
	var o database.Order
	if err := s.db.Where("id = ? and customer_id = ?", orderID, customerID).First(&o).Error; err != nil {
		return nil, errors.New("NOT_FOUND:order")
	}
	return &o, nil
}
func (s *Service) ListAdminOrders() ([]database.Order, error) {
	var orders []database.Order
	return orders, s.db.Order("created_at desc").Find(&orders).Error
}
func (s *Service) GetAdminOrder(orderID uuid.UUID) (*database.Order, error) {
	var o database.Order
	if err := s.db.Where("id = ?", orderID).First(&o).Error; err != nil {
		return nil, errors.New("NOT_FOUND:order")
	}
	return &o, nil
}

var allowedTransitions = map[string]map[string]bool{StatusPendingPayment: {StatusPaid: true, StatusCancelled: true, StatusExpired: true}, StatusPaid: {StatusProcessing: true, StatusCancelled: true}, StatusProcessing: {StatusReadyToShip: true}, StatusReadyToShip: {StatusShipped: true}, StatusShipped: {StatusCompleted: true}}

func (s *Service) UpdateOrderStatus(orderID uuid.UUID, newStatus string) (*database.Order, error) {
	newStatus = strings.ToUpper(strings.TrimSpace(newStatus))
	var o database.Order
	if err := s.db.Where("id = ?", orderID).First(&o).Error; err != nil {
		return nil, errors.New("NOT_FOUND:order")
	}
	if o.Status == newStatus {
		return &o, nil
	}
	if !allowedTransitions[o.Status][newStatus] {
		return nil, fmt.Errorf("VALIDATION:invalid status transition from %s to %s", o.Status, newStatus)
	}
	o.Status = newStatus
	if err := s.db.Save(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
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
