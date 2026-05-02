package payment

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go-be-mono-commerce/pkg/response"
)

type Handler struct{ svc PaymentService }

func NewHandler(svc PaymentService) *Handler { return &Handler{svc: svc} }

func (h *Handler) CreatePayment(c *gin.Context) {
	pay, created, err := h.svc.CreatePaymentForOrder(c.Request.Context(), c.Param("order_id"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Bad Request", "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"payment_id": pay.ID, "provider_reference": created.ProviderReference, "redirect_url": created.RedirectURL, "status": created.Status})
}

func (h *Handler) GetPaymentStatus(c *gin.Context) {
	st, err := h.svc.GetPaymentStatus(c.Request.Context(), c.Param("id"))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "Not Found", "NOT_FOUND", err.Error())
		return
	}
	response.OK(c, st)
}

func (h *Handler) MidtransWebhook(c *gin.Context) { h.handleWebhook(c, "midtrans") }
func (h *Handler) XenditWebhook(c *gin.Context)   { h.handleWebhook(c, "xendit") }

func (h *Handler) handleWebhook(c *gin.Context, provider string) {
	payload, _ := io.ReadAll(c.Request.Body)
	headers := map[string]string{}
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}
	if err := h.svc.HandleWebhook(c.Request.Context(), provider, headers, payload); err != nil {
		response.Fail(c, http.StatusBadRequest, "Bad Request", "BAD_REQUEST", err.Error())
		return
	}
	response.OK(c, gin.H{"received": true})
}
