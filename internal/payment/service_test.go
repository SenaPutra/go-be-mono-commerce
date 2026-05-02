package payment

import (
	"context"
	"testing"

	"go-be-mono-commerce/internal/config"
	"go-be-mono-commerce/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func mkCfg(provider string) config.Config {
	return config.Config{PaymentProvider: provider, PaymentMockMode: true, XenditCallbackToken: "token"}
}

func TestNewProviderFromEnv(t *testing.T) {
	for _, tc := range []struct {
		in string
		ok bool
	}{{"midtrans", true}, {"xendit", true}, {"unknown", false}} {
		p, err := NewProviderFromEnv(mkCfg(tc.in), tc.in)
		if tc.ok && (err != nil || p == nil) {
			t.Fatalf("expected provider")
		}
		if !tc.ok && err == nil {
			t.Fatalf("expected error")
		}
	}
}

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&database.Order{}, &database.Payment{}, &database.PaymentWebhookEvent{}, &database.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreatePaymentAndDuplicate(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-1", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	if err := db.Create(&ord).Error; err != nil {
		t.Fatal(err)
	}
	svc, _ := NewService(db, mkCfg("midtrans"))
	p1, _, err := svc.CreatePaymentForOrder(context.Background(), ord.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	p2, _, err := svc.CreatePaymentForOrder(context.Background(), ord.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if p1.ID != p2.ID {
		t.Fatalf("expected existing pending payment")
	}
}

func TestWebhookPaidAndNoDowngrade(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-2", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 1000, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	paidPayload := []byte(`{"id":"evt-1","external_id":"ORD-2","status":"PAID"}`)
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, paidPayload); err != nil {
		t.Fatal(err)
	}
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, paidPayload); err != nil {
		t.Fatal(err)
	}
	failPayload := []byte(`{"id":"evt-2","external_id":"ORD-2","status":"FAILED"}`)
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, failPayload); err != nil {
		t.Fatal(err)
	}

	var updated database.Payment
	db.First(&updated, "id = ?", pay.ID)
	if updated.Status != StatusPaid {
		t.Fatalf("should remain paid")
	}
	var o database.Order
	db.First(&o, "id = ?", ord.ID)
	if o.Status != StatusPaid {
		t.Fatalf("order should be paid")
	}
}

func TestApplyWebhookStatus_Idempotent(t *testing.T) {
	status, changed := applyWebhookStatus(StatusPending, StatusPaid)
	if !changed || status != StatusPaid {
		t.Fatal("expected paid")
	}
	status, changed = applyWebhookStatus(status, StatusFailed)
	if changed || status != StatusPaid {
		t.Fatal("must not downgrade paid")
	}
}
