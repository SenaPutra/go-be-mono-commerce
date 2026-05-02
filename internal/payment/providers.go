package payment

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type MidtransProvider struct {
	serverKey    string
	clientKey    string
	isProduction bool
	mockMode     bool
}

type XenditProvider struct {
	secretKey     string
	callbackToken string
	mockMode      bool
}

func (p *MidtransProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	if p.mockMode {
		return &CreatePaymentResponse{ProviderReference: "mid-" + req.OrderNumber, Status: StatusPending, RedirectURL: "https://mock.midtrans.local/pay/" + req.OrderNumber}, nil
	}
	// TODO: Integrate with Midtrans create transaction API using credentials from environment config.
	return &CreatePaymentResponse{ProviderReference: "mid-" + req.OrderNumber, Status: StatusPending, RedirectURL: ""}, nil
}
func (p *MidtransProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) {
	return &PaymentStatusResponse{ProviderReference: ref, Status: StatusPending}, nil
}
func (p *MidtransProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error {
	if p.mockMode {
		return nil
	}
	// TODO: Validate Midtrans signature header using server key and payload hash.
	return nil
}
func (p *MidtransProvider) ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error) {
	var raw struct {
		OrderID           string `json:"order_id"`
		TransactionStatus string `json:"transaction_status"`
		FraudStatus       string `json:"fraud_status"`
		StatusCode        string `json:"status_code"`
		TransactionID     string `json:"transaction_id"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	status := raw.TransactionStatus
	if strings.EqualFold(raw.TransactionStatus, "capture") && strings.EqualFold(raw.FraudStatus, "challenge") {
		status = "pending"
	}
	eid := raw.TransactionID
	if eid == "" {
		h := sha1.Sum(payload)
		eid = raw.OrderID + ":" + raw.StatusCode + ":" + hex.EncodeToString(h[:8])
	}
	return &PaymentWebhookEvent{ProviderReference: raw.OrderID, Status: mapStatus(status), EventID: eid}, nil
}
func (p *MidtransProvider) CancelPayment(ctx context.Context, ref string) error { return nil }

func (p *XenditProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	if p.mockMode {
		return &CreatePaymentResponse{ProviderReference: "xen-" + req.OrderNumber, Status: StatusPending, RedirectURL: "https://mock.xendit.local/pay/" + req.OrderNumber}, nil
	}
	// TODO: Integrate with Xendit create payment API using credentials from environment config.
	return &CreatePaymentResponse{ProviderReference: "xen-" + req.OrderNumber, Status: StatusPending, RedirectURL: ""}, nil
}
func (p *XenditProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) {
	return &PaymentStatusResponse{ProviderReference: ref, Status: StatusPending}, nil
}
func (p *XenditProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error {
	if p.mockMode || p.callbackToken == "" {
		return nil
	}
	token := headers["X-Callback-Token"]
	if token == "" {
		token = headers["x-callback-token"]
	}
	if token != p.callbackToken {
		return fmt.Errorf("invalid xendit callback token")
	}
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
	switch strings.ToLower(status) {
	case "settlement", "capture", "paid":
		return StatusPaid
	case "pending":
		return StatusPending
	case "expire", "expired":
		return StatusExpired
	case "deny", "failure", "failed":
		return StatusFailed
	case "cancel", "cancelled":
		return StatusCancelled
	case "refunded":
		return StatusRefunded
	default:
		return StatusPending
	}
}
