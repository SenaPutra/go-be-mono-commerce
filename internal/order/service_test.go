package order

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:order_test.db?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})
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

func TestConcurrentCheckoutSingleStock(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	product := database.Product{Name: "P", Slug: uuid.NewString(), PriceAmount: 100, Stock: 1, IsActive: true}
	_ = db.Create(&product).Error
	mkCustomer := func() (uuid.UUID, string) {
		cust := uuid.New()
		addr := database.CustomerAddress{CustomerID: cust, ReceiverName: "A", Address: "J", City: "C", Province: "P", PostalCode: "1"}
		_ = db.Create(&addr).Error
		cart := database.Cart{CustomerID: cust, Status: "ACTIVE"}
		_ = db.Create(&cart).Error
		_ = db.Create(&database.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 1, PriceSnapshotAmount: product.PriceAmount}).Error
		return cust, addr.ID.String()
	}
	custA, addrA := mkCustomer()
	custB, addrB := mkCustomer()
	results := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := svc.Checkout(custA, CheckoutRequest{AddressID: addrA})
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := svc.Checkout(custB, CheckoutRequest{AddressID: addrB})
		results <- err
	}()
	wg.Wait()
	close(results)
	success, fail := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else {
			var sErr *InsufficientStockError
			if !errors.As(err, &sErr) {
				t.Fatalf("expected insufficient stock err, got %v", err)
			}
			fail++
		}
	}
	if success != 1 || fail != 1 {
		t.Fatalf("expected 1 success and 1 fail, got success=%d fail=%d", success, fail)
	}
	var p database.Product
	_ = db.First(&p, "id = ?", product.ID).Error
	if p.Stock != 0 {
		t.Fatalf("expected stock 0, got %d", p.Stock)
	}
	if p.Stock < 0 {
		t.Fatalf("stock negative")
	}
	var orderCount int64
	_ = db.Model(&database.Order{}).Count(&orderCount).Error
	if orderCount != 1 {
		t.Fatalf("expected 1 order, got %d", orderCount)
	}
}

func TestConcurrentCheckoutMany(t *testing.T) {
	db := testDB(t)
	svc := NewService(db)
	product := database.Product{Name: "P2", Slug: uuid.NewString(), PriceAmount: 100, Stock: 5, IsActive: true}
	_ = db.Create(&product).Error
	const n = 10
	results := make(chan error, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		cust := uuid.New()
		addr := database.CustomerAddress{CustomerID: cust, ReceiverName: "A", Address: "J", City: "C", Province: "P", PostalCode: "1"}
		_ = db.Create(&addr).Error
		cart := database.Cart{CustomerID: cust, Status: "ACTIVE"}
		_ = db.Create(&cart).Error
		_ = db.Create(&database.CartItem{CartID: cart.ID, ProductID: product.ID, Quantity: 1, PriceSnapshotAmount: product.PriceAmount}).Error
		wg.Add(1)
		go func(c uuid.UUID, a string) {
			defer wg.Done()
			_, err := svc.Checkout(c, CheckoutRequest{AddressID: a})
			results <- err
		}(cust, addr.ID.String())
	}
	wg.Wait()
	close(results)
	success, fail := 0, 0
	for err := range results {
		if err == nil {
			success++
		} else {
			var sErr *InsufficientStockError
			if !errors.As(err, &sErr) {
				t.Fatalf("unexpected err: %v", err)
			}
			fail++
		}
	}
	if success != 5 || fail != 5 {
		t.Fatalf("expected 5 success/5 fail, got %d/%d", success, fail)
	}
	var p database.Product
	_ = db.First(&p, "id = ?", product.ID).Error
	if p.Stock != 0 {
		t.Fatalf("expected stock 0 got %d", p.Stock)
	}
	if p.Stock < 0 {
		t.Fatalf("stock negative")
	}
}
