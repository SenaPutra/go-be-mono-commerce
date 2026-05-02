package audit

import (
	"testing"

	"go-be-mono-commerce/internal/database"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestListAuditLogs(t *testing.T) {
	db, _ := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	db.AutoMigrate(&database.AuditLog{})
	db.Create(&database.AuditLog{ActorType: "admin", Action: "LOGIN", ResourceType: "auth"})
	db.Create(&database.AuditLog{ActorType: "system", Action: "WEBHOOK", ResourceType: "payment"})
	svc := NewService(db)
	f := Filter{ActorType: "admin", Page: 1, Limit: 10}
	items, total, err := svc.List(f)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatal("unexpected result")
	}
}
