package payment

import (
	"context"
	"encoding/json"
)

type MidtransProvider struct{}
type XenditProvider struct{}

func (p *MidtransProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// TODO: Integrate with Midtrans create transaction API using credentials from environment config.
	return &CreatePaymentResponse{ProviderReference: "mid-" + req.OrderNumber, Status: StatusPending, RedirectURL: "https://example.com/midtrans"}, nil
}
func (p *MidtransProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) {
	// TODO: Integrate with Midtrans get transaction status API.
	return &PaymentStatusResponse{ProviderReference: ref, Status: StatusPending}, nil
}
func (p *MidtransProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error {
	// TODO: Validate Midtrans signature header using server key.
	return nil
}
func (p *MidtransProvider) ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error) {
	var raw struct {
		OrderID           string `json:"order_id"`
		TransactionStatus string `json:"transaction_status"`
		StatusCode        string `json:"status_code"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	return &PaymentWebhookEvent{ProviderReference: raw.OrderID, Status: mapStatus(raw.TransactionStatus), EventID: raw.OrderID + ":" + raw.StatusCode}, nil
}
func (p *MidtransProvider) CancelPayment(ctx context.Context, ref string) error { return nil }

func (p *XenditProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// TODO: Integrate with Xendit create payment API using credentials from environment config.
	return &CreatePaymentResponse{ProviderReference: "xen-" + req.OrderNumber, Status: StatusPending, RedirectURL: "https://example.com/xendit"}, nil
}
func (p *XenditProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) {
	// TODO: Integrate with Xendit get payment status API.
	return &PaymentStatusResponse{ProviderReference: ref, Status: StatusPending}, nil
}
func (p *XenditProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error {
	// TODO: Validate Xendit webhook signature using callback token/secret.
	return nil
}
func (p *XenditProvider) ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error) {
	var raw struct {
		ExternalID string `json:"external_id"`
		Status     string `json:"status"`
		ID         string `json:"id"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	return &PaymentWebhookEvent{ProviderReference: raw.ExternalID, Status: mapStatus(raw.Status), EventID: raw.ID}, nil
}
func (p *XenditProvider) CancelPayment(ctx context.Context, ref string) error { return nil }

func mapStatus(status string) string {
	switch status {
	case "settlement", "paid", "PAID":
		return StatusPaid
	case "expire", "expired", "EXPIRED":
		return StatusExpired
	case "deny", "failed", "FAILED":
		return StatusFailed
	case "cancel", "cancelled", "CANCELLED":
		return StatusCancelled
	case "refunded", "REFUNDED":
		return StatusRefunded
	default:
		return StatusPending
	}
}
