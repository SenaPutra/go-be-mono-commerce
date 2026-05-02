package database

import (
	"go-be-mono-commerce/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, db.AutoMigrate(&Customer{}, &CustomerAddress{}, &AdminUser{}, &Category{}, &Product{}, &ProductImage{}, &Cart{}, &CartItem{}, &Order{}, &OrderItem{}, &Payment{}, &AuditLog{})
}
