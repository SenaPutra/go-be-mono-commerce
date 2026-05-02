package payment

import (
	"context"
	"sync"
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
	if err := db.AutoMigrate(&database.Product{}, &database.Order{}, &database.OrderItem{}, &database.Payment{}, &database.PaymentWebhookEvent{}, &database.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestWebhookExpiredRestoresStockIdempotent(t *testing.T) {
	db := setupDB(t)
	product := database.Product{Name: "P", Slug: "p-1", PriceAmount: 100, Stock: 3, IsActive: true}
	db.Create(&product)
	ord := database.Order{OrderNumber: "ORD-EXP", TotalAmount: 200, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	db.Create(&database.OrderItem{OrderID: ord.ID, ProductID: product.ID, Quantity: 2, PriceAmount: 100, SubtotalAmount: 200, ProductNameSnapshot: "P"})
	db.Model(&database.Product{}).Where("id = ?", product.ID).Update("stock", 1)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 200, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	payload := []byte(`{"id":"evt-exp-1","external_id":"ORD-EXP","status":"EXPIRED"}`)
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, payload); err != nil {
		t.Fatal(err)
	}
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, payload); err != nil {
		t.Fatal(err)
	}
	var updatedProduct database.Product
	db.First(&updatedProduct, "id = ?", product.ID)
	if updatedProduct.Stock != 3 {
		t.Fatalf("expected stock restored once to 3, got %d", updatedProduct.Stock)
	}
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
	status, changed := applyWebhookStatus(StatusPending, StatusPaid, "PENDING_PAYMENT")
	if !changed || status != StatusPaid {
		t.Fatal("expected paid")
	}
	status, changed = applyWebhookStatus(status, StatusFailed, "PAID")
	if changed || status != StatusPaid {
		t.Fatal("must not downgrade paid")
	}
}

func TestWebhookDuplicatePaidSequential(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-SEQ", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 1000, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	payload := []byte(`{"id":"evt-seq-1","external_id":"ORD-SEQ","status":"PAID"}`)
	for range 2 {
		if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, payload); err != nil {
			t.Fatal(err)
		}
	}
	var cnt int64
	db.Model(&database.PaymentWebhookEvent{}).Where("provider = ? AND event_id = ?", "xendit", "evt-seq-1").Count(&cnt)
	if cnt != 1 {
		t.Fatalf("expected single webhook event row, got %d", cnt)
	}
}

func TestWebhookDuplicatePaidConcurrent(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-CON", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 1000, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	payload := []byte(`{"id":"evt-con-1","external_id":"ORD-CON","status":"PAID"}`)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, payload)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var cnt int64
	db.Model(&database.PaymentWebhookEvent{}).Where("provider = ? AND event_id = ?", "xendit", "evt-con-1").Count(&cnt)
	if cnt != 1 {
		t.Fatalf("expected single event row, got %d", cnt)
	}
}

func TestWebhookPaidThenExpiredLateNoDowngrade(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-LATE", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 1000, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, []byte(`{"id":"evt-late-1","external_id":"ORD-LATE","status":"PAID"}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, []byte(`{"id":"evt-late-2","external_id":"ORD-LATE","status":"EXPIRED"}`)); err != nil {
		t.Fatal(err)
	}
	var updated database.Payment
	db.First(&updated, "id = ?", pay.ID)
	if updated.Status != StatusPaid {
		t.Fatalf("expected paid to be terminal, got %s", updated.Status)
	}
}

func TestWebhookFailedThenPaidAllowed(t *testing.T) {
	db := setupDB(t)
	ord := database.Order{OrderNumber: "ORD-FP", TotalAmount: 1000, Status: "PENDING_PAYMENT"}
	db.Create(&ord)
	pay := database.Payment{OrderID: ord.ID, Provider: "xendit", ProviderReference: ord.OrderNumber, Amount: 1000, Status: StatusPending}
	db.Create(&pay)
	svc, _ := NewService(db, mkCfg("xendit"))
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, []byte(`{"id":"evt-fp-1","external_id":"ORD-FP","status":"FAILED"}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.HandleWebhook(context.Background(), "xendit", map[string]string{"X-Callback-Token": "token"}, []byte(`{"id":"evt-fp-2","external_id":"ORD-FP","status":"PAID"}`)); err != nil {
		t.Fatal(err)
	}
	var updated database.Payment
	db.First(&updated, "id = ?", pay.ID)
	if updated.Status != StatusPaid {
		t.Fatalf("expected transition FAILED -> PAID when order not cancelled, got %s", updated.Status)
	}
}
