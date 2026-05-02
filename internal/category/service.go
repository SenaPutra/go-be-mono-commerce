package category

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/pkg/response"
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type UpsertCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (s *Service) ListActive() ([]database.Category, error) {
	var out []database.Category
	return out, s.db.Where("is_active = ?", true).Order("name asc").Find(&out).Error
}

func (s *Service) Create(req UpsertCategoryRequest) (*database.Category, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return nil, errors.New("VALIDATION:name and slug are required")
	}
	c := &database.Category{Name: req.Name, Slug: strings.ToLower(strings.TrimSpace(req.Slug)), IsActive: true}
	if err := s.db.Create(c).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("CONFLICT:category slug already exists")
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) Update(id uuid.UUID, req UpsertCategoryRequest) (*database.Category, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Slug) == "" {
		return nil, errors.New("VALIDATION:name and slug are required")
	}
	var c database.Category
	if err := s.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, errors.New("NOT_FOUND")
	}
	c.Name = req.Name
	c.Slug = strings.ToLower(strings.TrimSpace(req.Slug))
	if err := s.db.Save(&c).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("CONFLICT:category slug already exists")
		}
		return nil, err
	}
	return &c, nil
}

func (s *Service) Delete(id uuid.UUID) error {
	res := s.db.Delete(&database.Category{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("NOT_FOUND")
	}
	return nil
}

func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
	rg.GET("/categories", func(c *gin.Context) {
		data, err := svc.ListActive()
		if err != nil {
			response.Fail(c, 500, "Internal server error", "INTERNAL_ERROR", nil)
			return
		}
		response.OK(c, gin.H{"items": data})
	})
}
