package customer

import (
	"errors"
	"strings"

	"github.com/google/uuid"
	"go-be-mono-commerce/internal/database"
	"gorm.io/gorm"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type UpsertAddressRequest struct {
	ReceiverName string `json:"receiver_name"`
	Phone        string `json:"phone"`
	Address      string `json:"address"`
	City         string `json:"city"`
	Province     string `json:"province"`
	PostalCode   string `json:"postal_code"`
	IsDefault    bool   `json:"is_default"`
}

func (s *Service) GetProfile(customerID uuid.UUID) (*database.Customer, error) {
	return s.repo.GetCustomerByID(customerID)
}

func (s *Service) UpdateProfile(customerID uuid.UUID, req UpdateProfileRequest) (*database.Customer, error) {
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Phone) == "" {
		return nil, errors.New("VALIDATION:name and phone are required")
	}
	c, err := s.repo.GetCustomerByID(customerID)
	if err != nil {
		return nil, err
	}
	c.Name = strings.TrimSpace(req.Name)
	c.Phone = strings.TrimSpace(req.Phone)
	if err := s.repo.UpdateCustomer(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *Service) ListAddresses(customerID uuid.UUID) ([]database.CustomerAddress, error) {
	return s.repo.ListAddresses(customerID)
}

func validateAddress(req UpsertAddressRequest) error {
	if strings.TrimSpace(req.ReceiverName) == "" || strings.TrimSpace(req.Phone) == "" || strings.TrimSpace(req.Address) == "" || strings.TrimSpace(req.City) == "" || strings.TrimSpace(req.Province) == "" || strings.TrimSpace(req.PostalCode) == "" {
		return errors.New("VALIDATION:receiver_name, phone, address, city, province, postal_code are required")
	}
	return nil
}

func (s *Service) CreateAddress(customerID uuid.UUID, req UpsertAddressRequest) (*database.CustomerAddress, error) {
	if err := validateAddress(req); err != nil {
		return nil, err
	}
	addr := &database.CustomerAddress{CustomerID: customerID, ReceiverName: strings.TrimSpace(req.ReceiverName), Phone: strings.TrimSpace(req.Phone), Address: strings.TrimSpace(req.Address), City: strings.TrimSpace(req.City), Province: strings.TrimSpace(req.Province), PostalCode: strings.TrimSpace(req.PostalCode), IsDefault: req.IsDefault}
	if req.IsDefault {
		if err := s.repo.UnsetDefaultAddresses(customerID); err != nil {
			return nil, err
		}
	}
	if err := s.repo.CreateAddress(addr); err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *Service) UpdateAddress(customerID, addressID uuid.UUID, req UpsertAddressRequest) (*database.CustomerAddress, error) {
	if err := validateAddress(req); err != nil {
		return nil, err
	}
	addr, err := s.repo.GetAddressByID(customerID, addressID)
	if err != nil {
		return nil, errors.New("NOT_FOUND:address")
	}
	if req.IsDefault {
		if err := s.repo.UnsetDefaultAddresses(customerID); err != nil {
			return nil, err
		}
	}
	addr.ReceiverName = strings.TrimSpace(req.ReceiverName)
	addr.Phone = strings.TrimSpace(req.Phone)
	addr.Address = strings.TrimSpace(req.Address)
	addr.City = strings.TrimSpace(req.City)
	addr.Province = strings.TrimSpace(req.Province)
	addr.PostalCode = strings.TrimSpace(req.PostalCode)
	addr.IsDefault = req.IsDefault
	if err := s.repo.UpdateAddress(addr); err != nil {
		return nil, err
	}
	return addr, nil
}

func (s *Service) DeleteAddress(customerID, addressID uuid.UUID) error {
	_, err := s.repo.GetAddressByID(customerID, addressID)
	if err != nil {
		return errors.New("NOT_FOUND:address")
	}
	return s.repo.DeleteAddress(customerID, addressID)
}

func (s *Service) ListOwnOrders(customerID uuid.UUID) ([]database.Order, error) {
	return s.repo.ListCustomerOrders(customerID)
}
func (s *Service) GetOwnOrder(customerID, orderID uuid.UUID) (*database.Order, error) {
	o, err := s.repo.GetCustomerOrderByID(customerID, orderID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("NOT_FOUND:order")
	}
	return o, err
}
func (s *Service) AdminListCustomers() ([]database.Customer, error) { return s.repo.ListCustomers() }
func (s *Service) AdminGetCustomer(customerID uuid.UUID) (*database.Customer, error) {
	return s.repo.GetCustomerByID(customerID)
}
func (s *Service) AdminListCustomerOrders(customerID uuid.UUID) ([]database.Order, error) {
	return s.repo.ListCustomerOrders(customerID)
}

func HandleErr(err error) (int, string, string, []string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 404, "Not found", "NOT_FOUND", []string{"resource not found"}
	}
	m := err.Error()
	if strings.HasPrefix(m, "VALIDATION:") {
		return 400, "Validation error", "VALIDATION_ERROR", []string{strings.TrimPrefix(m, "VALIDATION:")}
	}
	if strings.HasPrefix(m, "NOT_FOUND:") {
		return 404, "Not found", "NOT_FOUND", []string{strings.TrimPrefix(m, "NOT_FOUND:") + " not found"}
	}
	return 500, "Internal server error", "INTERNAL_ERROR", nil
}
