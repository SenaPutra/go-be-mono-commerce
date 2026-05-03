package report

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&database.Order{}, &database.OrderItem{}, &database.Product{}, &database.Payment{}, &database.AuditLog{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestParseFilterDateRangeValidation(t *testing.T) {
	_, err := ParseFilter("2026-02-01", "2026-01-01", "1", "10")
	if err == nil {
		t.Fatal("expected error for invalid date range")
	}
}

func seedReportData(t *testing.T, db *gorm.DB) {
	prod := database.Product{Name: "P1", Slug: "p1", PriceAmount: 1000, Stock: 10, IsActive: true}
	if err := db.Create(&prod).Error; err != nil {
		t.Fatal(err)
	}
	statuses := []string{"PENDING_PAYMENT", "PAID", "PROCESSING", "SHIPPED", "COMPLETED", "CANCELLED", "EXPIRED", "FAILED"}
	for i, st := range statuses {
		o := database.Order{CustomerID: uuid.New(), OrderNumber: "ORD-" + uuid.NewString(), TotalAmount: int64((i + 1) * 100), Status: st}
		if err := db.Create(&o).Error; err != nil {
			t.Fatal(err)
		}
		if st == "PAID" || st == "COMPLETED" {
			oi := database.OrderItem{OrderID: o.ID, ProductID: prod.ID, ProductNameSnapshot: "P1", Quantity: 2, PriceAmount: 100, SubtotalAmount: 200}
			db.Create(&oi)
		}
	}
	pays := []database.Payment{{OrderID: uuid.New(), Provider: "midtrans", Status: "PENDING", Amount: 100}, {OrderID: uuid.New(), Provider: "midtrans", Status: "PAID", Amount: 200}, {OrderID: uuid.New(), Provider: "xendit", Status: "FAILED", Amount: 300}, {OrderID: uuid.New(), Provider: "xendit", Status: "EXPIRED", Amount: 400}}
	for _, p := range pays {
		db.Create(&p)
	}
}

func TestReportsAggregation(t *testing.T) {
	db := setupDB(t)
	seedReportData(t, db)
	svc := NewService(db)
	f := Filter{Page: 1, Limit: 10}
	ord, _ := svc.OrderReport(f)
	if ord.TotalOrders != 8 || ord.TotalPaid != 1 || ord.TotalFailed != 1 {
		t.Fatal("invalid order report")
	}
	sales, _ := svc.SalesReport(f)
	if sales.GrossSalesAmount != 3600 || sales.PaidSalesAmount != 200 || sales.CompletedSalesAmount != 500 {
		t.Fatal("invalid sales report")
	}
	if sales.AverageOrderValue != 450 {
		t.Fatal("invalid average order value")
	}
	products, _, _ := svc.ProductSalesReport(f)
	if len(products) != 1 || products[0].TotalQuantitySold != 4 {
		t.Fatal("invalid product report")
	}
	pay, _ := svc.PaymentReport(f)
	if pay.TotalPayments != 4 || pay.TotalPaid != 1 || pay.TotalByProvider["midtrans"] != 2 {
		t.Fatal("invalid payment report")
	}
}

func TestReportsDateFilter(t *testing.T) {
	db := setupDB(t)
	seedReportData(t, db)
	svc := NewService(db)

	today := time.Now().Format("2006-01-02")
	f, err := ParseFilter(today, today, "1", "10")
	if err != nil {
		t.Fatal(err)
	}
	ord, err := svc.OrderReport(f)
	if err != nil {
		t.Fatal(err)
	}
	if ord.TotalOrders == 0 {
		t.Fatal("expected records for today's date filter")
	}
}
