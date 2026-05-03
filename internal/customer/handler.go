package customer

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go-be-mono-commerce/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func userIDFromContext(c *gin.Context) (uuid.UUID, error) { return uuid.Parse(c.GetString("user_id")) }

func (h *Handler) Me(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	out, err := h.svc.GetProfile(uid)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, out)
}
func (h *Handler) UpdateMe(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	var req UpdateProfileRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
		return
	}
	out, err := h.svc.UpdateProfile(uid, req)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, out)
}
func (h *Handler) ListAddresses(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	out, err := h.svc.ListAddresses(uid)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, gin.H{"items": out})
}
func (h *Handler) CreateAddress(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	var req UpsertAddressRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
		return
	}
	out, err := h.svc.CreateAddress(uid, req)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.Created(c, out)
}
func (h *Handler) UpdateAddress(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
		return
	}
	var req UpsertAddressRequest
	if c.ShouldBindJSON(&req) != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid request body"})
		return
	}
	out, err := h.svc.UpdateAddress(uid, aid, req)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, out)
}
func (h *Handler) DeleteAddress(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
		return
	}
	if err := h.svc.DeleteAddress(uid, aid); err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}
func (h *Handler) ListMyOrders(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	out, err := h.svc.ListOwnOrders(uid)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, gin.H{"items": out})
}
func (h *Handler) GetMyOrder(c *gin.Context) {
	uid, err := userIDFromContext(c)
	if err != nil {
		response.Fail(c, 401, "Unauthorized", "UNAUTHORIZED", nil)
		return
	}
	oid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
		return
	}
	out, err := h.svc.GetOwnOrder(uid, oid)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, out)
}
func (h *Handler) AdminListCustomers(c *gin.Context) {
	out, err := h.svc.AdminListCustomers()
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, gin.H{"items": out})
}
func (h *Handler) AdminGetCustomer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
		return
	}
	out, err := h.svc.AdminGetCustomer(id)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, out)
}
func (h *Handler) AdminGetCustomerOrders(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Fail(c, 400, "Validation error", "VALIDATION_ERROR", []string{"invalid id"})
		return
	}
	out, err := h.svc.AdminListCustomerOrders(id)
	if err != nil {
		code, msg, ec, de := HandleErr(err)
		response.Fail(c, code, msg, ec, de)
		return
	}
	response.OK(c, gin.H{"items": out})
}
