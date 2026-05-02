package audit

import (
	"time"

	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/pkg/pagination"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Filter struct {
	ActorType    string
	Action       string
	ResourceType string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	Limit        int
}

func ParseFilter(actorType, action, resourceType, dateFrom, dateTo, page, limit string) (Filter, error) {
	f := Filter{ActorType: actorType, Action: action, ResourceType: resourceType}
	if dateFrom != "" {
		t, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			return f, err
		}
		f.DateFrom = &t
	}
	if dateTo != "" {
		t, err := time.Parse("2006-01-02", dateTo)
		if err != nil {
			return f, err
		}
		eod := t.Add(24*time.Hour - time.Nanosecond)
		f.DateTo = &eod
	}
	f.Page, f.Limit = pagination.Parse(page, limit)
	return f, nil
}

func (s *Service) List(f Filter) ([]database.AuditLog, int64, error) {
	q := s.db.Model(&database.AuditLog{})
	if f.ActorType != "" {
		q = q.Where("actor_type = ?", f.ActorType)
	}
	if f.Action != "" {
		q = q.Where("action = ?", f.Action)
	}
	if f.ResourceType != "" {
		q = q.Where("resource_type = ?", f.ResourceType)
	}
	if f.DateFrom != nil {
		q = q.Where("created_at >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where("created_at <= ?", *f.DateTo)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []database.AuditLog
	err := q.Order("created_at DESC").Offset((f.Page - 1) * f.Limit).Limit(f.Limit).Find(&items).Error
	return items, total, err
}
