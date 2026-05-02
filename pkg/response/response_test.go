package response

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	apperrors "go-be-mono-commerce/pkg/errors"
)

func TestErrorMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"validation", apperrors.Validation([]string{"bad"}), 400, "VALIDATION_ERROR"},
		{"unauthorized", apperrors.New(apperrors.CodeUnauthorized, "Unauthorized", nil), 401, "UNAUTHORIZED"},
		{"forbidden", apperrors.New(apperrors.CodeForbidden, "Forbidden", nil), 403, "FORBIDDEN"},
		{"not found", apperrors.New(apperrors.CodeNotFound, "Not found", nil), 404, "NOT_FOUND"},
		{"conflict", apperrors.New(apperrors.CodeConflict, "Conflict", nil), 409, "CONFLICT"},
		{"insufficient stock", apperrors.New(apperrors.CodeInsufficientStock, "Conflict", nil), 409, "INSUFFICIENT_STOCK"},
		{"invalid transition", apperrors.New(apperrors.CodeInvalidStatusTransition, "Conflict", nil), 409, "INVALID_STATUS_TRANSITION"},
		{"duplicate webhook", apperrors.New(apperrors.CodeDuplicateWebhookEvent, "Conflict", nil), 409, "DUPLICATE_WEBHOOK_EVENT"},
		{"idempotency", apperrors.New(apperrors.CodeIdempotencyConflict, "Conflict", nil), 409, "IDEMPOTENCY_CONFLICT"},
		{"not implemented", apperrors.New(apperrors.CodeNotImplemented, "Not implemented", nil), 501, "NOT_IMPLEMENTED"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			Error(c, tc.err)
			if w.Code != tc.status {
				t.Fatalf("expected status %d got %d", tc.status, w.Code)
			}
			var body struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != tc.code {
				t.Fatalf("expected code %s got %s", tc.code, body.Error.Code)
			}
		})
	}
}
