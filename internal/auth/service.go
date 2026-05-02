package auth

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"go-be-mono-commerce/pkg/response"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	RoleCustomer = "CUSTOMER"
	RoleAdmin    = "ADMIN"
)

var emailRegex = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

type Service struct {
	db        *gorm.DB
	jwtSecret string
	jwtTTL    time.Duration
}

func NewService(db *gorm.DB, jwtSecret, ttlHoursRaw string) *Service {
	ttlHours := 24
	if v, err := strconv.Atoi(ttlHoursRaw); err == nil && v > 0 {
		ttlHours = v
	}
	return &Service{db: db, jwtSecret: jwtSecret, jwtTTL: time.Duration(ttlHours) * time.Hour}
}

type RegisterCustomerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func validateEmailPassword(email, password string) []string {
	var errs []string
	if !emailRegex.MatchString(strings.TrimSpace(email)) {
		errs = append(errs, "email must be a valid email address")
	}
	if len(password) < 8 {
		errs = append(errs, "password must be at least 8 characters")
	}
	return errs
}

func (s *Service) RegisterCustomer(req RegisterCustomerRequest) (*database.Customer, error) {
	if errs := validateEmailPassword(req.Email, req.Password); len(errs) > 0 {
		return nil, errors.New("VALIDATION:" + strings.Join(errs, ";"))
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	c := &database.Customer{Name: req.Name, Email: strings.ToLower(strings.TrimSpace(req.Email)), Phone: req.Phone, PasswordHash: string(hash), IsActive: true}
	if err := s.db.Create(c).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, errors.New("CONFLICT:email already registered")
		}
		return nil, err
	}
	return c, nil
}

func (s *Service) LoginCustomer(req LoginRequest) (string, *database.Customer, error) {
	if errs := validateEmailPassword(req.Email, req.Password); len(errs) > 0 {
		return "", nil, errors.New("VALIDATION:" + strings.Join(errs, ";"))
	}
	var c database.Customer
	if err := s.db.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&c).Error; err != nil {
		return "", nil, errors.New("UNAUTHORIZED")
	}
	if !c.IsActive {
		return "", nil, errors.New("INACTIVE")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(req.Password)); err != nil {
		return "", nil, errors.New("UNAUTHORIZED")
	}
	token, err := SignWithTTL(s.jwtSecret, c.ID.String(), RoleCustomer, s.jwtTTL)
	return token, &c, err
}

func (s *Service) LoginAdmin(req LoginRequest) (string, *database.AdminUser, error) {
	if errs := validateEmailPassword(req.Email, req.Password); len(errs) > 0 {
		return "", nil, errors.New("VALIDATION:" + strings.Join(errs, ";"))
	}
	var a database.AdminUser
	if err := s.db.Where("email = ?", strings.ToLower(strings.TrimSpace(req.Email))).First(&a).Error; err != nil {
		return "", nil, errors.New("UNAUTHORIZED")
	}
	if !a.IsActive {
		return "", nil, errors.New("INACTIVE")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(req.Password)); err != nil {
		return "", nil, errors.New("UNAUTHORIZED")
	}
	token, err := SignWithTTL(s.jwtSecret, a.ID.String(), RoleAdmin, s.jwtTTL)
	return token, &a, err
}

func RegisterRoutes(rg *gin.RouterGroup, svc *Service) {
	rg.POST("/customer/register", func(c *gin.Context) {
		var req RegisterCustomerRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, http.StatusBadRequest, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		customer, err := svc.RegisterCustomer(req)
		if err != nil {
			handleAuthErr(c, err)
			return
		}
		response.Created(c, gin.H{"id": customer.ID, "name": customer.Name, "email": customer.Email, "phone": customer.Phone})
	})
	rg.POST("/customer/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, http.StatusBadRequest, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		token, customer, err := svc.LoginCustomer(req)
		if err != nil {
			handleAuthErr(c, err)
			return
		}
		response.OK(c, gin.H{"access_token": token, "token_type": "Bearer", "user": gin.H{"id": customer.ID, "name": customer.Name, "email": customer.Email, "role": RoleCustomer}})
	})
	rg.POST("/admin/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, http.StatusBadRequest, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
			return
		}
		token, admin, err := svc.LoginAdmin(req)
		if err != nil {
			handleAuthErr(c, err)
			return
		}
		response.OK(c, gin.H{"access_token": token, "token_type": "Bearer", "user": gin.H{"id": admin.ID, "name": admin.Name, "email": admin.Email, "role": admin.Role}})
	})
}

func handleAuthErr(c *gin.Context, err error) {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "VALIDATION:"):
		response.Fail(c, http.StatusBadRequest, "Validation error", "VALIDATION_ERROR", strings.Split(strings.TrimPrefix(msg, "VALIDATION:"), ";"))
	case strings.HasPrefix(msg, "CONFLICT:"):
		response.Fail(c, http.StatusConflict, "Conflict", "CONFLICT", []string{strings.TrimPrefix(msg, "CONFLICT:")})
	case msg == "INACTIVE":
		response.Fail(c, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED", []string{"user is inactive"})
	case msg == "UNAUTHORIZED":
		response.Fail(c, http.StatusUnauthorized, "Unauthorized", "UNAUTHORIZED", []string{"invalid credentials"})
	default:
		response.Fail(c, http.StatusInternalServerError, "Internal server error", "INTERNAL_ERROR", nil)
	}
}

func ParseUUID(id string) (uuid.UUID, error) { return uuid.Parse(id) }
