package product

import (
	"errors"
	"math"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/pkg/pagination"
	"go-be-mono-commerce/pkg/response"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type ProductImageInput struct {
	ImageURL  string `json:"image_url"`
	IsPrimary bool   `json:"is_primary"`
}
type UpsertProductRequest struct {
	CategoryID  string              `json:"category_id"`
	Name        string              `json:"name"`
	Slug        string              `json:"slug"`
	Description string              `json:"description"`
	PriceAmount int64               `json:"price_amount"`
	Stock       int                 `json:"stock"`
	Images      []ProductImageInput `json:"images"`
}

type ProductListResponse struct {
	Items      []gin.H `json:"items"`
	Page       int     `json:"page"`
	Limit      int     `json:"limit"`
	Total      int64   `json:"total"`
	TotalPages int     `json:"total_pages"`
}

func validate(req UpsertProductRequest) error {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" || strings.TrimSpace(req.CategoryID) == "" {
		return errors.New("VALIDATION:name, slug, and category_id are required")
	}
	if req.PriceAmount < 0 {
		return errors.New("VALIDATION:price_amount must be >= 0")
	}
	if req.Stock < 0 {
		return errors.New("VALIDATION:stock must be >= 0")
	}
	return nil
}

func (s *Service) Create(req UpsertProductRequest) (*database.Product, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, errors.New("VALIDATION:category_id must be valid uuid")
	}
	var cat database.Category
	if err := s.db.First(&cat, "id = ?", cid).Error; err != nil {
		return nil, errors.New("VALIDATION:category not found")
	}
	p := &database.Product{CategoryID: &cid, Name: req.Name, Slug: strings.ToLower(strings.TrimSpace(req.Slug)), Description: req.Description, PriceAmount: req.PriceAmount, Stock: req.Stock}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(p).Error; err != nil {
			return err
		}
		for _, img := range req.Images {
			if strings.TrimSpace(img.ImageURL) == "" {
				continue
			}
			if err := tx.Create(&database.ProductImage{ProductID: p.ID, ImageURL: img.ImageURL, IsPrimary: img.IsPrimary}).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("CONFLICT:product slug already exists")
		}
		return nil, err
	}
	return p, nil
}
func (s *Service) Update(id uuid.UUID, req UpsertProductRequest) (*database.Product, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(req.CategoryID)
	if err != nil {
		return nil, errors.New("VALIDATION:category_id must be valid uuid")
	}
	var p database.Product
	if err := s.db.First(&p, "id = ?", id).Error; err != nil {
		return nil, errors.New("NOT_FOUND")
	}
	p.CategoryID = &cid
	p.Name = req.Name
	p.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	p.Description = req.Description
	p.PriceAmount = req.PriceAmount
	p.Stock = req.Stock
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&p).Error; err != nil {
			return err
		}
		if err := tx.Where("product_id = ?", p.ID).Delete(&database.ProductImage{}).Error; err != nil {
			return err
		}
		for _, img := range req.Images {
			if strings.TrimSpace(img.ImageURL) == "" {
				continue
			}
			if err := tx.Create(&database.ProductImage{ProductID: p.ID, ImageURL: img.ImageURL, IsPrimary: img.IsPrimary}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("CONFLICT:product slug already exists")
		}
		return nil, err
	}
	return &p, nil
}
func (s *Service) Delete(id uuid.UUID) error {
	res := s.db.Delete(&database.Product{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}
	return nil
}
func (s *Service) SetActive(id uuid.UUID, active bool) error {
	res := s.db.Model(&database.Product{}).Where("id = ?", id).Update("is_active", active)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}
	return nil
}
func (s *Service) SetStock(id uuid.UUID, stock int) error {
	if stock < 0 {
		return errors.New("VALIDATION:stock must be >= 0")
	}
	res := s.db.Model(&database.Product{}).Where("id = ?", id).Update("stock", stock)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}
	return nil
}

func (s *Service) ListPublic(c *gin.Context) (*ProductListResponse, error) {
	page, limit := pagination.Parse(c.Query("page"), c.Query("limit"))
	q := s.db.Model(&database.Product{}).Where("is_active = ?", true)
	if cs := c.Query("category_slug"); cs != "" {
		q = q.Joins("JOIN categories on categories.id=products.category_id").Where("categories.slug = ?", strings.ToLower(cs))
	}
	if s1 := c.Query("search"); s1 != "" {
		like := "%" + strings.ToLower(s1) + "%"
		q = q.Where("lower(products.name) like ? OR lower(products.description) like ?", like, like)
	}
	if v := c.Query("min_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			q = q.Where("price_amount >= ?", n)
		}
	}
	if v := c.Query("max_price"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			q = q.Where("price_amount <= ?", n)
		}
	}
	sortBy := c.DefaultQuery("sort_by", "created_at")
	if sortBy != "price_amount" && sortBy != "name" && sortBy != "created_at" {
		sortBy = "created_at"
	}
	sortOrder := strings.ToUpper(c.DefaultQuery("sort_order", "DESC"))
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, err
	}
	var products []database.Product
	if err := q.Preload("", func(tx *gorm.DB) *gorm.DB { return tx }).Order(sortBy + " " + sortOrder).Offset((page - 1) * limit).Limit(limit).Find(&products).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(products))
	for _, p := range products {
		ids = append(ids, p.ID)
	}
	var images []database.ProductImage
	if len(ids) > 0 {
		_ = s.db.Where("product_id in ?", ids).Find(&images).Error
	}
	imgMap := map[uuid.UUID][]database.ProductImage{}
	for _, im := range images {
		imgMap[im.ProductID] = append(imgMap[im.ProductID], im)
	}
	items := make([]gin.H, 0, len(products))
	for _, p := range products {
		items = append(items, gin.H{"id": p.ID, "category_id": p.CategoryID, "name": p.Name, "slug": p.Slug, "description": p.Description, "price_amount": p.PriceAmount, "stock": p.Stock, "is_active": p.IsActive, "images": imgMap[p.ID]})
	}
	return &ProductListResponse{Items: items, Page: page, Limit: limit, Total: total, TotalPages: int(math.Ceil(float64(total) / float64(limit)))}, nil
}

func (s *Service) DetailBySlug(slug string) (gin.H, error) {
	var p database.Product
	if err := s.db.Where("slug = ? and is_active = ?", strings.ToLower(slug), true).First(&p).Error; err != nil {
		return nil, errors.New("NOT_FOUND")
	}
	var cat database.Category
	_ = s.db.First(&cat, "id = ?", p.CategoryID).Error
	var images []database.ProductImage
	_ = s.db.Where("product_id = ?", p.ID).Find(&images).Error
	return gin.H{"id": p.ID, "name": p.Name, "slug": p.Slug, "description": p.Description, "price_amount": p.PriceAmount, "stock": p.Stock, "is_active": p.IsActive, "category": cat, "images": images}, nil
}

func HandleErr(c *gin.Context, err error) {
	m := err.Error()
	switch {
	case strings.HasPrefix(m, "VALIDATION:"):
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{strings.TrimPrefix(m, "VALIDATION:")})
	case strings.HasPrefix(m, "CONFLICT:"):
		response.Fail(c, 409, "Conflict", "CONFLICT", []string{strings.TrimPrefix(m, "CONFLICT:")})
	case m == "NOT_FOUND":
		response.Fail(c, 404, "Not found", "NOT_FOUND", []string{"resource not found"})
	default:
		response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
	}
}
