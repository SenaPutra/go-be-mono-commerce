package order

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
	if err := db.AutoMigrate(&database.Product{}, &database.Cart{}, &database.CartItem{}, &database.Order{}, &database.OrderItem{}, &database.CustomerAddress{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedCart(t *testing.T, db *gorm.DB, cust uuid.UUID, qty1, stock1, qty2, stock2 int) (database.Product, database.Product) {
	cart := database.Cart{CustomerID: cust, Status: "ACTIVE"}
	_ = db.Create(&cart).Error
	p1 := database.Product{Name: "P1", Slug: uuid.NewString(), PriceAmount: 100, Stock: stock1, IsActive: true}
	p2 := database.Product{Name: "P2", Slug: uuid.NewString(), PriceAmount: 50, Stock: stock2, IsActive: true}
	_ = db.Create(&p1).Error
	_ = db.Create(&p2).Error
	_ = db.Create(&database.CartItem{CartID: cart.ID, ProductID: p1.ID, Quantity: qty1, PriceSnapshotAmount: p1.PriceAmount}).Error
	_ = db.Create(&database.CartItem{CartID: cart.ID, ProductID: p2.ID, Quantity: qty2, PriceSnapshotAmount: p2.PriceAmount}).Error
	return p1, p2
}

func TestCheckoutSuccess(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	cust := uuid.New()
	addr := database.CustomerAddress{CustomerID: cust, ReceiverName: "A", Address: "J", City: "C", Province: "P", PostalCode: "1"}
	_ = db.Create(&addr).Error
	p1, _ := seedCart(t, db, cust, 2, 10, 1, 10)
	o, err := svc.Checkout(cust, CheckoutRequest{AddressID: addr.ID.String()})
	if err != nil {
		t.Fatal(err)
	}
	var items []database.OrderItem
	_ = db.Where("order_id = ?", o.ID).Find(&items).Error
	if len(items) != 2 {
		t.Fatalf("expected 2 items")
	}
	var p database.Product
	_ = db.First(&p, "id = ?", p1.ID).Error
	if p.Stock != 8 {
		t.Fatalf("stock not reduced")
	}
}

func TestCheckoutEmptyCart(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	cust := uuid.New()
	addr := database.CustomerAddress{CustomerID: cust, ReceiverName: "A", Address: "J", City: "C", Province: "P", PostalCode: "1"}
	_ = db.Create(&addr).Error
	_ = db.Create(&database.Cart{CustomerID: cust, Status: "ACTIVE"}).Error
	if _, err := svc.Checkout(cust, CheckoutRequest{AddressID: addr.ID.String()}); err == nil {
		t.Fatal("expected err")
	}
}

func TestCheckoutInsufficientStockRollback(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	cust := uuid.New()
	addr := database.CustomerAddress{CustomerID: cust, ReceiverName: "A", Address: "J", City: "C", Province: "P", PostalCode: "1"}
	_ = db.Create(&addr).Error
	p1, p2 := seedCart(t, db, cust, 2, 10, 5, 2)
	if _, err := svc.Checkout(cust, CheckoutRequest{AddressID: addr.ID.String()}); err == nil {
		t.Fatal("expected err")
	}
	var c int64
	_ = db.Model(&database.Order{}).Count(&c).Error
	if c != 0 {
		t.Fatalf("order should rollback")
	}
	var rp1, rp2 database.Product
	_ = db.First(&rp1, "id = ?", p1.ID).Error
	_ = db.First(&rp2, "id = ?", p2.ID).Error
	if rp1.Stock != 10 || rp2.Stock != 2 {
		t.Fatalf("stock should rollback")
	}
}

func TestInvalidOrderTransition(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	o := database.Order{CustomerID: uuid.New(), OrderNumber: "ORD-1", TotalAmount: 1, Status: StatusCompleted}
	_ = db.Create(&o).Error
	if _, err := svc.UpdateOrderStatus(o.ID, StatusProcessing); err == nil {
		t.Fatal("expected invalid transition")
	}
}
