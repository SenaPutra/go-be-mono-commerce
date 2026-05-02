package product

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&database.Category{}, &database.Product{}, &database.ProductImage{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestCreateProductPricingValidation(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	cat := database.Category{Name: "Audio", Slug: "audio"}
	_ = db.Create(&cat).Error

	if _, err := svc.Create(UpsertProductRequest{CategoryID: cat.ID.String(), Name: "P1", Slug: "p1", PriceAmount: 100, Stock: 1}); err != nil {
		t.Fatalf("expected no compare_at_price_amount to pass: %v", err)
	}
	compare := int64(150)
	if _, err := svc.Create(UpsertProductRequest{CategoryID: cat.ID.String(), Name: "P2", Slug: "p2", PriceAmount: 100, CompareAtPriceAmount: &compare, IsDiscountActive: true, Stock: 1}); err != nil {
		t.Fatalf("expected valid compare_at_price_amount to pass: %v", err)
	}
	low := int64(90)
	if _, err := svc.Create(UpsertProductRequest{CategoryID: cat.ID.String(), Name: "P3", Slug: "p3", PriceAmount: 100, CompareAtPriceAmount: &low, Stock: 1}); err == nil {
		t.Fatal("expected lower compare_at_price_amount validation error")
	}
	equal := int64(100)
	if _, err := svc.Create(UpsertProductRequest{CategoryID: cat.ID.String(), Name: "P4", Slug: "p4", PriceAmount: 100, CompareAtPriceAmount: &equal, Stock: 1}); err == nil {
		t.Fatal("expected equal compare_at_price_amount validation error")
	}
}

func TestDiscountDisplayRules(t *testing.T) {
	now := time.Now()
	compare := int64(350000)
	p := database.Product{ID: uuid.New(), PriceAmount: 250000, CompareAtPriceAmount: &compare, IsDiscountActive: true}
	shown, pct := discountDisplay(p, now)
	if !shown || pct != 28.57 {
		t.Fatalf("unexpected discount shown=%v pct=%v", shown, pct)
	}

	p.IsDiscountActive = false
	shown, pct = discountDisplay(p, now)
	if shown || pct != 0 {
		t.Fatal("expected hidden discount when promo inactive")
	}

	p.IsDiscountActive = true
	start := now.Add(2 * time.Hour)
	end := now.Add(3 * time.Hour)
	p.DiscountStartAt = &start
	p.DiscountEndAt = &end
	shown, pct = discountDisplay(p, now)
	if shown || pct != 0 {
		t.Fatal("expected hidden discount when outside promo period")
	}
}
