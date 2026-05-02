package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/gorm"
)

const (
	StatusPending   = "PENDING"
	StatusPaid      = "PAID"
	StatusExpired   = "EXPIRED"
	StatusFailed    = "FAILED"
	StatusCancelled = "CANCELLED"
	StatusRefunded  = "REFUNDED"
)

type PaymentService interface {
	CreatePaymentForOrder(ctx context.Context, orderID string) (*database.Payment, *CreatePaymentResponse, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatusResponse, error)
	HandleWebhook(ctx context.Context, provider string, headers map[string]string, payload []byte) error
}

type paymentService struct {
	db        *gorm.DB
	provider  PaymentProvider
	providers map[string]PaymentProvider
}

func NewProviderFromEnv(providerName string) (PaymentProvider, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "midtrans":
		return &MidtransProvider{}, nil
	case "xendit":
		return &XenditProvider{}, nil
	default:
		return nil, fmt.Errorf("unsupported payment provider: %s", providerName)
	}
}

func NewService(db *gorm.DB, providerName string) (PaymentService, error) {
	selected, err := NewProviderFromEnv(providerName)
	if err != nil {
		return nil, err
	}
	return &paymentService{db: db, provider: selected, providers: map[string]PaymentProvider{"midtrans": &MidtransProvider{}, "xendit": &XenditProvider{}}}, nil
}

func (s *paymentService) CreatePaymentForOrder(ctx context.Context, orderID string) (*database.Payment, *CreatePaymentResponse, error) {
	var order database.Order
	if err := s.db.WithContext(ctx).First(&order, "id = ?", orderID).Error; err != nil {
		return nil, nil, err
	}
	if order.Status == StatusPaid {
		return nil, nil, errors.New("order already paid")
	}

	req := CreatePaymentRequest{OrderID: order.ID.String(), OrderNumber: order.OrderNumber, Amount: order.TotalAmount}
	created, err := s.provider.CreatePayment(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	pay := &database.Payment{OrderID: order.ID, Provider: providerName(s.provider), ProviderReference: created.ProviderReference, Status: created.Status, Amount: order.TotalAmount}
	if err := s.db.WithContext(ctx).Create(pay).Error; err != nil {
		return nil, nil, err
	}
	return pay, created, nil
}

func (s *paymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*PaymentStatusResponse, error) {
	var pay database.Payment
	if err := s.db.WithContext(ctx).First(&pay, "id = ?", paymentID).Error; err != nil {
		return nil, err
	}
	return &PaymentStatusResponse{ProviderReference: pay.ProviderReference, Status: pay.Status}, nil
}

func (s *paymentService) HandleWebhook(ctx context.Context, provider string, headers map[string]string, payload []byte) error {
	p, ok := s.providers[strings.ToLower(provider)]
	if !ok {
		return errors.New("unknown provider")
	}
	if err := p.ValidateWebhook(ctx, headers, payload); err != nil {
		return err
	}
	evt, err := p.ParseWebhook(ctx, payload)
	if err != nil {
		return err
	}
	meta, _ := json.Marshal(map[string]any{"provider": provider, "event": evt, "headers": headers})

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pay database.Payment
		if err := tx.Clauses().First(&pay, "provider = ? AND provider_reference = ?", strings.ToLower(provider), evt.ProviderReference).Error; err != nil {
			return err
		}
		nextStatus, changed := applyWebhookStatus(pay.Status, evt.Status)
		if !changed {
			return tx.Create(&database.AuditLog{Action: "PAYMENT_WEBHOOK_DUPLICATE", ResourceType: "payment", ResourceID: pay.ID.String(), Metadata: string(meta)}).Error
		}
		if err := tx.Model(&pay).Update("status", nextStatus).Error; err != nil {
			return err
		}
		var newOrderStatus string
		switch nextStatus {
		case StatusPaid:
			newOrderStatus = StatusPaid
		case StatusExpired:
			newOrderStatus = StatusExpired
		case StatusFailed:
			newOrderStatus = StatusFailed
		}
		if newOrderStatus != "" {
			if err := tx.Model(&database.Order{}).Where("id = ?", pay.OrderID).Update("status", newOrderStatus).Error; err != nil {
				return err
			}
		}
		return tx.Create(&database.AuditLog{Action: "PAYMENT_WEBHOOK", ResourceType: "payment", ResourceID: pay.ID.String(), Metadata: string(meta)}).Error
	})
}

func providerName(p PaymentProvider) string {
	switch p.(type) {
	case *MidtransProvider:
		return "midtrans"
	case *XenditProvider:
		return "xendit"
	default:
		return "unknown"
	}
}

func ParseUUID(v string) (uuid.UUID, error) { return uuid.Parse(v) }

func applyWebhookStatus(current, incoming string) (string, bool) {
	if current == incoming {
		return current, false
	}
	return incoming, true
}
