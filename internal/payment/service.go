package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/database"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func NewProviderFromEnv(cfg config.Config, providerName string) (PaymentProvider, error) {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case "midtrans":
		return &MidtransProvider{serverKey: cfg.MidtransServerKey, clientKey: cfg.MidtransClientKey, isProduction: cfg.MidtransIsProduction, mockMode: cfg.PaymentMockMode}, nil
	case "xendit":
		return &XenditProvider{secretKey: cfg.XenditSecretKey, callbackToken: cfg.XenditCallbackToken, mockMode: cfg.PaymentMockMode}, nil
	default:
		return nil, fmt.Errorf("unsupported payment provider: %s", providerName)
	}
}

func NewService(db *gorm.DB, cfg config.Config) (PaymentService, error) {
	selected, err := NewProviderFromEnv(cfg, cfg.PaymentProvider)
	if err != nil {
		return nil, err
	}
	mid, _ := NewProviderFromEnv(cfg, "midtrans")
	xen, _ := NewProviderFromEnv(cfg, "xendit")
	return &paymentService{db: db, provider: selected, providers: map[string]PaymentProvider{"midtrans": mid, "xendit": xen}}, nil
}

func (s *paymentService) CreatePaymentForOrder(ctx context.Context, orderID string) (*database.Payment, *CreatePaymentResponse, error) {
	var pay *database.Payment
	var created *CreatePaymentResponse
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var order database.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, "id = ?", orderID).Error; err != nil {
			return err
		}
		if order.Status != "PENDING_PAYMENT" {
			return errors.New("order is not in pending payment status")
		}
		var existing database.Payment
		err := tx.Where("order_id = ? AND status = ?", order.ID, StatusPending).Order("created_at desc").First(&existing).Error
		if err == nil {
			pay = &existing
			created = &CreatePaymentResponse{ProviderReference: existing.ProviderReference, RedirectURL: existing.RedirectURL, Status: existing.Status}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		req := CreatePaymentRequest{OrderID: order.ID.String(), OrderNumber: order.OrderNumber, Amount: order.TotalAmount}
		c, err := s.provider.CreatePayment(ctx, req)
		if err != nil {
			return err
		}
		pay = &database.Payment{OrderID: order.ID, Provider: providerName(s.provider), ProviderReference: c.ProviderReference, Status: StatusPending, Amount: order.TotalAmount, RedirectURL: c.RedirectURL}
		if err := tx.Create(pay).Error; err != nil {
			return err
		}
		created = c
		return nil
	})
	return pay, created, err
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
	now := time.Now().UTC()

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		event := database.PaymentWebhookEvent{Provider: strings.ToLower(provider), EventID: evt.EventID, ProviderReference: evt.ProviderReference, Status: evt.Status, RawPayload: datatypes.JSON(payload)}
		if err := tx.Where("provider = ? AND event_id = ?", event.Provider, event.EventID).FirstOrCreate(&event).Error; err != nil {
			return err
		}
		if event.ProcessedAt != nil {
			return s.writeAudit(tx, "PAYMENT_WEBHOOK_DUPLICATE", provider, evt, headers, payload)
		}

		var pay database.Payment
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&pay, "provider = ? AND provider_reference = ?", strings.ToLower(provider), evt.ProviderReference).Error; err != nil {
			return err
		}
		nextStatus, changed := applyWebhookStatus(pay.Status, evt.Status)
		if changed {
			updates := map[string]any{"status": nextStatus}
			if nextStatus == StatusPaid && pay.PaidAt == nil {
				updates["paid_at"] = now
			}
			if err := tx.Model(&pay).Updates(updates).Error; err != nil {
				return err
			}
			if nextStatus == StatusPaid || nextStatus == StatusExpired || nextStatus == StatusFailed {
				var ord database.Order
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ord, "id = ?", pay.OrderID).Error; err != nil {
					return err
				}
				if err := restoreOrderStockIfNeeded(tx, &ord, nextStatus); err != nil {
					return err
				}
				if err := tx.Model(&ord).Updates(map[string]any{"status": nextStatus, "stock_restored": ord.StockRestored}).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&event).Update("processed_at", now).Error; err != nil {
			return err
		}
		if !changed {
			return s.writeAudit(tx, "PAYMENT_WEBHOOK_DUPLICATE", provider, evt, headers, payload)
		}
		return s.writeAudit(tx, "PAYMENT_WEBHOOK", provider, evt, headers, payload)
	})
}

func (s *paymentService) writeAudit(tx *gorm.DB, action, provider string, evt *PaymentWebhookEvent, headers map[string]string, payload []byte) error {
	meta, _ := json.Marshal(map[string]any{"provider": provider, "event": evt, "headers": headers, "payload": json.RawMessage(payload)})
	return tx.Create(&database.AuditLog{ActorType: "SYSTEM", Action: action, ResourceType: "payment_webhook", Metadata: datatypes.JSON(meta)}).Error
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

func applyWebhookStatus(current, incoming string) (string, bool) {
	if current == StatusPaid && incoming != StatusPaid {
		return current, false
	}
	if current == incoming {
		return current, false
	}
	return incoming, true
}

func restoreOrderStockIfNeeded(tx *gorm.DB, order *database.Order, nextStatus string) error {
	if order.Status != "PENDING_PAYMENT" {
		return nil
	}
	if nextStatus != StatusExpired && nextStatus != StatusFailed && nextStatus != StatusCancelled {
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
