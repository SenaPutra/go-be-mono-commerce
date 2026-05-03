package customer

import (
	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/gorm"
)

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) GetCustomerByID(id uuid.UUID) (*database.Customer, error) {
	var c database.Customer
	if err := r.db.Where("id = ?", id).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) UpdateCustomer(c *database.Customer) error {
	return r.db.Model(&database.Customer{}).Where("id = ?", c.ID).Updates(map[string]interface{}{"name": c.Name, "phone": c.Phone}).Error
}

func (r *Repository) ListAddresses(customerID uuid.UUID) ([]database.CustomerAddress, error) {
	var items []database.CustomerAddress
	return items, r.db.Where("customer_id = ?", customerID).Order("created_at desc").Find(&items).Error
}

func (r *Repository) CreateAddress(addr *database.CustomerAddress) error {
	return r.db.Create(addr).Error
}

func (r *Repository) GetAddressByID(customerID, addressID uuid.UUID) (*database.CustomerAddress, error) {
	var a database.CustomerAddress
	if err := r.db.Where("id = ? AND customer_id = ?", addressID, customerID).First(&a).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) UpdateAddress(addr *database.CustomerAddress) error {
	return r.db.Save(addr).Error
}

func (r *Repository) DeleteAddress(customerID, addressID uuid.UUID) error {
	return r.db.Where("id = ? AND customer_id = ?", addressID, customerID).Delete(&database.CustomerAddress{}).Error
}

func (r *Repository) UnsetDefaultAddresses(customerID uuid.UUID) error {
	return r.db.Model(&database.CustomerAddress{}).Where("customer_id = ?", customerID).Update("is_default", false).Error
}

func (r *Repository) ListCustomers() ([]database.Customer, error) {
	var items []database.Customer
	return items, r.db.Order("created_at desc").Find(&items).Error
}

func (r *Repository) ListCustomerOrders(customerID uuid.UUID) ([]database.Order, error) {
	var items []database.Order
	return items, r.db.Where("customer_id = ?", customerID).Order("created_at desc").Find(&items).Error
}

func (r *Repository) GetCustomerOrderByID(customerID, orderID uuid.UUID) (*database.Order, error) {
	var o database.Order
	if err := r.db.Where("id = ? AND customer_id = ?", orderID, customerID).First(&o).Error; err != nil {
		return nil, err
	}
	return &o, nil
}
