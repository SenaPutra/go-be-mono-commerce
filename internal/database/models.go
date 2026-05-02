package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type Customer struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(150);not null" json:"name"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	Phone        string    `gorm:"type:varchar(50)" json:"phone"`
	PasswordHash string    `gorm:"type:text;not null" json:"-"`
	IsActive     bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CustomerAddress struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID   uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	ReceiverName string    `gorm:"type:varchar(150);not null" json:"receiver_name"`
	Phone        string    `gorm:"type:varchar(50)" json:"phone"`
	Address      string    `gorm:"type:text;not null" json:"address"`
	City         string    `gorm:"type:varchar(100);not null" json:"city"`
	Province     string    `gorm:"type:varchar(100);not null" json:"province"`
	PostalCode   string    `gorm:"type:varchar(20);not null" json:"postal_code"`
	IsDefault    bool      `gorm:"not null;default:false" json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type AdminUser struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(150);not null" json:"name"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	PasswordHash string    `gorm:"type:text;not null" json:"-"`
	Role         string    `gorm:"type:varchar(50);not null;default:'SUPER_ADMIN'" json:"role"`
	IsActive     bool      `gorm:"not null;default:true" json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Category struct {
	BaseModel
	Name     string `gorm:"type:varchar(150);not null" json:"name"`
	Slug     string `gorm:"type:varchar(180);not null;uniqueIndex" json:"slug"`
	IsActive bool   `gorm:"not null;default:true" json:"is_active"`
}

type Product struct {
	BaseModel
	CategoryID           *uuid.UUID `gorm:"type:uuid;index" json:"category_id"`
	Name                 string     `gorm:"type:varchar(200);not null" json:"name"`
	Slug                 string     `gorm:"type:varchar(220);not null;uniqueIndex" json:"slug"`
	Description          string     `gorm:"type:text" json:"description"`
	PriceAmount          int64      `gorm:"not null" json:"price_amount"`
	CompareAtPriceAmount *int64     `gorm:"default:null" json:"compare_at_price_amount"`
	DiscountStartAt      *time.Time `json:"discount_start_at"`
	DiscountEndAt        *time.Time `json:"discount_end_at"`
	IsDiscountActive     bool       `gorm:"not null;default:false" json:"is_discount_active"`
	Stock                int        `gorm:"not null;default:0" json:"stock"`
	IsActive             bool       `gorm:"not null;default:false" json:"is_active"`
}

type ProductImage struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductID uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	ImageURL  string    `gorm:"type:text;not null" json:"image_url"`
	IsPrimary bool      `gorm:"not null;default:false" json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Cart struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	Status     string    `gorm:"type:varchar(50);not null;default:'ACTIVE'" json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CartItem struct {
	ID                  uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CartID              uuid.UUID `gorm:"type:uuid;not null;index" json:"cart_id"`
	ProductID           uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	Quantity            int       `gorm:"not null" json:"quantity"`
	PriceSnapshotAmount int64     `gorm:"not null" json:"price_snapshot_amount"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Order struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	CustomerID  uuid.UUID `gorm:"type:uuid;not null;index" json:"customer_id"`
	OrderNumber string    `gorm:"type:varchar(80);not null;uniqueIndex" json:"order_number"`
	TotalAmount int64     `gorm:"not null" json:"total_amount"`
	Status      string    `gorm:"type:varchar(50);not null;default:'PENDING_PAYMENT'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type OrderItem struct {
	ID                  uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID             uuid.UUID `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID           uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductNameSnapshot string    `gorm:"type:varchar(200);not null" json:"product_name_snapshot"`
	Quantity            int       `gorm:"not null" json:"quantity"`
	PriceAmount         int64     `gorm:"not null" json:"price_amount"`
	SubtotalAmount      int64     `gorm:"not null" json:"subtotal_amount"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Payment struct {
	ID                uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OrderID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"order_id"`
	Provider          string         `gorm:"type:varchar(50);not null;index" json:"provider"`
	ProviderReference string         `gorm:"type:varchar(255);index" json:"provider_reference"`
	PaymentMethod     string         `gorm:"type:varchar(100)" json:"payment_method"`
	Amount            int64          `gorm:"not null" json:"amount"`
	Status            string         `gorm:"type:varchar(50);not null;default:'PENDING'" json:"status"`
	RedirectURL       string         `gorm:"type:text" json:"redirect_url"`
	PaidAt            *time.Time     `json:"paid_at"`
	RawPayload        datatypes.JSON `gorm:"type:jsonb" json:"raw_payload"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type PaymentWebhookEvent struct {
	ID                uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Provider          string         `gorm:"type:varchar(50);not null" json:"provider"`
	EventID           string         `gorm:"type:varchar(255);not null" json:"event_id"`
	ProviderReference string         `gorm:"type:varchar(255);index" json:"provider_reference"`
	Status            string         `gorm:"type:varchar(50);not null" json:"status"`
	RawPayload        datatypes.JSON `gorm:"type:jsonb" json:"raw_payload"`
	ProcessedAt       *time.Time     `json:"processed_at"`
	CreatedAt         time.Time      `json:"created_at"`
}

type AuditLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ActorID      *uuid.UUID     `gorm:"type:uuid" json:"actor_id"`
	ActorType    string         `gorm:"type:varchar(100);not null" json:"actor_type"`
	Action       string         `gorm:"type:varchar(100);not null" json:"action"`
	ResourceType string         `gorm:"type:varchar(100);not null" json:"resource_type"`
	ResourceID   *uuid.UUID     `gorm:"type:uuid" json:"resource_id"`
	Metadata     datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}
