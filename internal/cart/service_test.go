package cart

import (
	"testing"

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
	if err := db.AutoMigrate(&database.Product{}, &database.Cart{}, &database.CartItem{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAddUpdateClearCart(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	custID := uuid.New()
	p := database.Product{Name: "P1", Slug: "p1", PriceAmount: 1000, Stock: 5, IsActive: true}
	if err := db.Create(&p).Error; err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(custID, AddItemRequest{ProductID: p.ID.String(), Quantity: 2}); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItem(custID, AddItemRequest{ProductID: p.ID.String(), Quantity: 1}); err != nil {
		t.Fatal(err)
	}
	cart, err := svc.Get(custID)
	if err != nil {
		t.Fatal(err)
	}
	items := cart["items"].([]map[string]interface{})
	if len(items) != 1 || items[0]["quantity"].(int) != 3 {
		t.Fatalf("unexpected items: %#v", items)
	}
	itemID := items[0]["id"].(uuid.UUID)
	if err := svc.UpdateItem(custID, itemID, 4); err != nil {
		t.Fatal(err)
	}
	if err := svc.Clear(custID); err != nil {
		t.Fatal(err)
	}
	cart, _ = svc.Get(custID)
	if len(cart["items"].([]map[string]interface{})) != 0 {
		t.Fatal("expected empty")
	}
}

func TestAddItemValidation(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	custID := uuid.New()
	p := database.Product{Name: "P2", Slug: "p2", PriceAmount: 1000, Stock: 1, IsActive: false}
	_ = db.Create(&p).Error
	if err := svc.AddItem(custID, AddItemRequest{ProductID: p.ID.String(), Quantity: 1}); err == nil {
		t.Fatal("expected inactive error")
	}
	p.IsActive = true
	_ = db.Save(&p).Error
	if err := svc.AddItem(custID, AddItemRequest{ProductID: p.ID.String(), Quantity: 2}); err == nil {
		t.Fatal("expected stock error")
	}
}
