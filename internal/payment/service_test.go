package payment

import "testing"

func TestNewProviderFromEnv(t *testing.T) {
	tests := []struct{ in string; ok bool }{{"midtrans", true}, {"xendit", true}, {"unknown", false}}
	for _, tc := range tests {
		p, err := NewProviderFromEnv(tc.in)
		if tc.ok && (err != nil || p == nil) { t.Fatalf("expected provider for %s", tc.in) }
		if !tc.ok && err == nil { t.Fatalf("expected error for %s", tc.in) }
	}
}

func TestApplyWebhookStatus_Idempotent(t *testing.T) {
	status, changed := applyWebhookStatus(StatusPending, StatusPaid)
	if !changed || status != StatusPaid { t.Fatalf("expected changed to PAID") }
	status, changed = applyWebhookStatus(status, StatusPaid)
	if changed || status != StatusPaid { t.Fatalf("expected idempotent PAID") }
}
