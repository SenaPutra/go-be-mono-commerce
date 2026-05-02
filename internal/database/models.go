package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *BaseModel) BeforeCreate(*gorm.DB) error { b.ID = uuid.New(); return nil }

type Customer struct {
	BaseModel
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Name         string
}
type CustomerAddress struct {
	BaseModel
	CustomerID           uuid.UUID `gorm:"type:uuid;index"`
	Label, Address, City string
}
type AdminUser struct {
	BaseModel
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Name         string
}
type Category struct {
	BaseModel
	Name, Slug string `gorm:"index"`
	IsActive   bool
}
type Product struct {
	BaseModel
	CategoryID  uuid.UUID `gorm:"type:uuid;index"`
	Name, Slug  string    `gorm:"index"`
	Description string
	Price       int64
	Stock       int
	Published   bool
}
type ProductImage struct {
	BaseModel
	ProductID uuid.UUID `gorm:"type:uuid;index"`
	URL       string
}
type Cart struct {
	BaseModel
	CustomerID uuid.UUID `gorm:"type:uuid;index"`
	IsActive   bool
}
type CartItem struct {
	BaseModel
	CartID, ProductID uuid.UUID `gorm:"type:uuid;index"`
	Qty               int
	PriceSnapshot     int64
}
type Order struct {
	BaseModel
	CustomerID  uuid.UUID `gorm:"type:uuid;index"`
	OrderNumber string    `gorm:"index"`
	Status      string
	TotalAmount int64
}
type OrderItem struct {
	BaseModel
	OrderID, ProductID uuid.UUID `gorm:"type:uuid;index"`
	ProductName        string
	Qty                int
	Price              int64
}
type Payment struct {
	BaseModel
	OrderID                             uuid.UUID `gorm:"type:uuid;index"`
	Provider, ProviderReference, Status string    `gorm:"index"`
	Amount                              int64
}
type AuditLog struct {
	BaseModel
	ActorID                                    *uuid.UUID `gorm:"type:uuid"`
	Action, ResourceType, ResourceID, Metadata string
}
