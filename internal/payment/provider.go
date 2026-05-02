package payment

import "context"

type CreatePaymentRequest struct{ OrderID, OrderNumber string; Amount int64 }
type CreatePaymentResponse struct{ ProviderReference, RedirectURL, Status string }
type PaymentStatusResponse struct{ ProviderReference, Status string }
type PaymentWebhookEvent struct{ ProviderReference, Status, EventID string }

type PaymentProvider interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)
	GetPaymentStatus(ctx context.Context, providerReference string) (*PaymentStatusResponse, error)
	ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error
	ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error)
	CancelPayment(ctx context.Context, providerReference string) error
}
