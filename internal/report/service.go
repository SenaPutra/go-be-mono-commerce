package report

import (
	"time"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/pkg/pagination"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Filter struct {
	DateFrom *time.Time
	DateTo   *time.Time
	Page     int
	Limit    int
}

func ParseFilter(dateFrom, dateTo, page, limit string) (Filter, error) {
	f := Filter{}
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

type OrderReport struct {
	TotalOrders         int64 `json:"total_orders"`
	TotalPendingPayment int64 `json:"total_pending_payment"`
	TotalPaid           int64 `json:"total_paid"`
	TotalProcessing     int64 `json:"total_processing"`
	TotalShipped        int64 `json:"total_shipped"`
	TotalCompleted      int64 `json:"total_completed"`
	TotalCancelled      int64 `json:"total_cancelled"`
	TotalExpired        int64 `json:"total_expired"`
	TotalFailed         int64 `json:"total_failed"`
}

func (s *Service) OrderReport(f Filter) (OrderReport, error) {
	q := s.db.Model(&database.Order{})
	q = applyDateFilter(q, "created_at", f)
	var out OrderReport
	err := q.Select(`
		COUNT(*) as total_orders,
		SUM(CASE WHEN status = 'PENDING_PAYMENT' THEN 1 ELSE 0 END) as total_pending_payment,
		SUM(CASE WHEN status = 'PAID' THEN 1 ELSE 0 END) as total_paid,
		SUM(CASE WHEN status = 'PROCESSING' THEN 1 ELSE 0 END) as total_processing,
		SUM(CASE WHEN status = 'SHIPPED' THEN 1 ELSE 0 END) as total_shipped,
		SUM(CASE WHEN status = 'COMPLETED' THEN 1 ELSE 0 END) as total_completed,
		SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END) as total_cancelled,
		SUM(CASE WHEN status = 'EXPIRED' THEN 1 ELSE 0 END) as total_expired,
		SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as total_failed
	`).Scan(&out).Error
	return out, err
}

type SalesReport struct {
	GrossSalesAmount     int64   `json:"gross_sales_amount"`
	PaidSalesAmount      int64   `json:"paid_sales_amount"`
	CompletedSalesAmount int64   `json:"completed_sales_amount"`
	AverageOrderValue    float64 `json:"average_order_value"`
}

func (s *Service) SalesReport(f Filter) (SalesReport, error) {
	q := applyDateFilter(s.db.Model(&database.Order{}), "created_at", f)
	var out SalesReport
	err := q.Select(`
		COALESCE(SUM(total_amount), 0) as gross_sales_amount,
		COALESCE(SUM(CASE WHEN status = 'PAID' THEN total_amount ELSE 0 END), 0) as paid_sales_amount,
		COALESCE(SUM(CASE WHEN status = 'COMPLETED' THEN total_amount ELSE 0 END), 0) as completed_sales_amount,
		COALESCE(AVG(total_amount), 0) as average_order_value
	`).Scan(&out).Error
	return out, err
}

type ProductSalesItem struct {
	ProductID         uuid.UUID `json:"product_id"`
	ProductName       string    `json:"product_name"`
	TotalQuantitySold int64     `json:"total_quantity_sold"`
	TotalSalesAmount  int64     `json:"total_sales_amount"`
}

func (s *Service) ProductSalesReport(f Filter) ([]ProductSalesItem, int64, error) {
	base := s.db.Table("order_items oi").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("o.status IN ?", []string{"PAID", "PROCESSING", "READY_TO_SHIP", "SHIPPED", "COMPLETED"})
	base = applyDateFilter(base, "o.created_at", f)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ProductSalesItem
	err := base.Select("oi.product_id, p.name as product_name, SUM(oi.quantity) as total_quantity_sold, SUM(oi.subtotal_amount) as total_sales_amount").
		Group("oi.product_id, p.name").
		Order("total_quantity_sold DESC").
		Offset((f.Page - 1) * f.Limit).Limit(f.Limit).
		Scan(&items).Error
	return items, total, err
}

type PaymentReport struct {
	TotalPayments   int64            `json:"total_payments"`
	TotalPending    int64            `json:"total_pending"`
	TotalPaid       int64            `json:"total_paid"`
	TotalFailed     int64            `json:"total_failed"`
	TotalExpired    int64            `json:"total_expired"`
	TotalByProvider map[string]int64 `json:"total_by_provider"`
}

func (s *Service) PaymentReport(f Filter) (PaymentReport, error) {
	q := applyDateFilter(s.db.Model(&database.Payment{}), "created_at", f)
	var out PaymentReport
	if err := q.Select(`
		COUNT(*) as total_payments,
		SUM(CASE WHEN status = 'PENDING' THEN 1 ELSE 0 END) as total_pending,
		SUM(CASE WHEN status = 'PAID' THEN 1 ELSE 0 END) as total_paid,
		SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as total_failed,
		SUM(CASE WHEN status = 'EXPIRED' THEN 1 ELSE 0 END) as total_expired
	`).Scan(&out).Error; err != nil {
		return out, err
	}
	var rows []struct {
		Provider string
		Total    int64
	}
	if err := q.Select("provider, COUNT(*) as total").Group("provider").Scan(&rows).Error; err != nil {
		return out, err
	}
	out.TotalByProvider = map[string]int64{}
	for _, r := range rows {
		out.TotalByProvider[r.Provider] = r.Total
	}
	return out, nil
}

func applyDateFilter(q *gorm.DB, column string, f Filter) *gorm.DB {
	if f.DateFrom != nil {
		q = q.Where(column+" >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		q = q.Where(column+" <= ?", *f.DateTo)
	}
	return q
}
