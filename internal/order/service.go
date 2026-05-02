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
	StatusFailed         = "FAILED"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type CheckoutRequest struct {
	AddressID string `json:"address_id"`
	Notes     string `json:"notes"`
}

type InsufficientStockDetail struct {
	ProductID         uuid.UUID `json:"product_id"`
	RequestedQuantity int       `json:"requested_quantity"`
	AvailableStock    int       `json:"available_stock"`
}

type InsufficientStockError struct{ Detail InsufficientStockDetail }

func (e *InsufficientStockError) Error() string {
	return fmt.Sprintf("INSUFFICIENT_STOCK:product %s stock is %d, requested %d", e.Detail.ProductID, e.Detail.AvailableStock, e.Detail.RequestedQuantity)
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

		productSnapshots := make(map[uuid.UUID]database.Product, len(items))
		var total int64
		for _, it := range items {
			if it.Quantity <= 0 {
				return errors.New("VALIDATION:cart item quantity must be > 0")
			}
			var product database.Product
			if err := tx.Where("id = ?", it.ProductID).First(&product).Error; err != nil {
				return errors.New("NOT_FOUND:product")
			}
			if !product.IsActive {
				return errors.New("VALIDATION:product is inactive")
			}
			res := tx.Model(&database.Product{}).
				Where("id = ? AND stock >= ? AND deleted_at IS NULL AND is_active = ?", product.ID, it.Quantity, true).
				Update("stock", gorm.Expr("stock - ?", it.Quantity))
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				var latest database.Product
				if err := tx.Unscoped().Select("id", "stock").Where("id = ?", product.ID).First(&latest).Error; err != nil {
					return &InsufficientStockError{Detail: InsufficientStockDetail{ProductID: product.ID, RequestedQuantity: it.Quantity, AvailableStock: 0}}
				}
				return &InsufficientStockError{Detail: InsufficientStockDetail{ProductID: product.ID, RequestedQuantity: it.Quantity, AvailableStock: latest.Stock}}
			}
			productSnapshots[product.ID] = product
			sub := int64(it.Quantity) * it.PriceSnapshotAmount
			total += sub
		}

		order := database.Order{CustomerID: customerID, OrderNumber: generateOrderNumber(), TotalAmount: total, Status: StatusPendingPayment, StockRestored: false}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		for _, it := range items {
			if err := tx.Create(&database.OrderItem{OrderID: order.ID, ProductID: it.ProductID, ProductNameSnapshot: productSnapshots[it.ProductID].Name, Quantity: it.Quantity, PriceAmount: it.PriceSnapshotAmount, SubtotalAmount: int64(it.Quantity) * it.PriceSnapshotAmount}).Error; err != nil {
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

var allowedTransitions = map[string]map[string]bool{StatusPendingPayment: {StatusPaid: true, StatusCancelled: true, StatusExpired: true, StatusFailed: true}, StatusPaid: {StatusProcessing: true, StatusCancelled: true}, StatusProcessing: {StatusReadyToShip: true}, StatusReadyToShip: {StatusShipped: true}, StatusShipped: {StatusCompleted: true}}

func (s *Service) UpdateOrderStatus(orderID uuid.UUID, newStatus string) (*database.Order, error) {
	newStatus = strings.ToUpper(strings.TrimSpace(newStatus))
	var o database.Order
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", orderID).First(&o).Error; err != nil {
			return errors.New("NOT_FOUND:order")
		}
		if o.Status == newStatus {
			return nil
		}
		if !allowedTransitions[o.Status][newStatus] {
			return fmt.Errorf("VALIDATION:invalid status transition from %s to %s", o.Status, newStatus)
		}
		if err := maybeRestoreStock(tx, &o, o.Status, newStatus); err != nil {
			return err
		}
		o.Status = newStatus
		return tx.Save(&o).Error
	})
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func maybeRestoreStock(tx *gorm.DB, order *database.Order, oldStatus, newStatus string) error {
	if oldStatus != StatusPendingPayment {
		return nil
	}
	if newStatus != StatusExpired && newStatus != StatusFailed && newStatus != StatusCancelled {
		return nil
	}
	if order.StockRestored {
		return nil
	}
	var items []database.OrderItem
	if err := tx.Where("order_id = ?", order.ID).Find(&items).Error; err != nil {
		return err
	}
	for _, item := range items {
		if err := tx.Model(&database.Product{}).
			Where("id = ? AND deleted_at IS NULL", item.ProductID).
			Update("stock", gorm.Expr("stock + ?", item.Quantity)).Error; err != nil {
			return err
		}
	}
	order.StockRestored = true
	return nil
}

func HandleErr(err error) (int, string, string, []string) {
	var stockErr *InsufficientStockError
	if errors.As(err, &stockErr) {
		return 409, "Insufficient stock", "INSUFFICIENT_STOCK", []string{stockErr.Error()}
	}
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
