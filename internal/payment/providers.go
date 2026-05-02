package payment

import "context"

type MidtransProvider struct{}
type XenditProvider struct{}

func (p *MidtransProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) { return &CreatePaymentResponse{ProviderReference: "mid-" + req.OrderNumber, Status: "PENDING", RedirectURL: "https://example.com/midtrans"}, nil }
func (p *MidtransProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) { return &PaymentStatusResponse{ProviderReference: ref, Status: "PENDING"}, nil }
func (p *MidtransProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error { return nil }
func (p *MidtransProvider) ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error) { return &PaymentWebhookEvent{ProviderReference: "", Status: "PAID", EventID: "TODO"}, nil }
func (p *MidtransProvider) CancelPayment(ctx context.Context, ref string) error { return nil }

func (p *XenditProvider) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) { return &CreatePaymentResponse{ProviderReference: "xen-" + req.OrderNumber, Status: "PENDING", RedirectURL: "https://example.com/xendit"}, nil }
func (p *XenditProvider) GetPaymentStatus(ctx context.Context, ref string) (*PaymentStatusResponse, error) { return &PaymentStatusResponse{ProviderReference: ref, Status: "PENDING"}, nil }
func (p *XenditProvider) ValidateWebhook(ctx context.Context, headers map[string]string, payload []byte) error { return nil }
func (p *XenditProvider) ParseWebhook(ctx context.Context, payload []byte) (*PaymentWebhookEvent, error) { return &PaymentWebhookEvent{ProviderReference: "", Status: "PAID", EventID: "TODO"}, nil }
func (p *XenditProvider) CancelPayment(ctx context.Context, ref string) error { return nil }
