package response

import (
	"github.com/gin-gonic/gin"
	apperrors "go-be-mono-commerce/pkg/errors"
)

type ErrorBody struct {
	Code    string      `json:"code"`
	Details interface{} `json:"details"`
}

func JSON(c *gin.Context, code int, success bool, msg string, data interface{}, errBody *ErrorBody) {
	c.JSON(code, gin.H{"success": success, "message": msg, "data": data, "error": errBody})
}

func OK(c *gin.Context, data interface{})      { JSON(c, 200, true, "OK", data, nil) }
func Created(c *gin.Context, data interface{}) { JSON(c, 201, true, "Created", data, nil) }
func Fail(c *gin.Context, code int, msg, errCode string, details interface{}) {
	JSON(c, code, false, msg, nil, &ErrorBody{Code: errCode, Details: details})
}

func ValidationError(c *gin.Context, details interface{}) {
	Fail(c, 400, "Validation error", string(apperrors.CodeValidationError), details)
}

func Error(c *gin.Context, err error) {
	appErr := apperrors.AsAppError(err)
	code := httpStatus(appErr.Code)
	msg := appErr.Message
	if msg == "" {
		msg = "Internal server error"
	}
	Fail(c, code, msg, string(appErr.Code), appErr.Details)
}

func httpStatus(code apperrors.Code) int {
	switch code {
	case apperrors.CodeValidationError:
		return 400
	case apperrors.CodeUnauthorized:
		return 401
	case apperrors.CodeForbidden:
		return 403
	case apperrors.CodeNotFound:
		return 404
	case apperrors.CodeConflict, apperrors.CodeInsufficientStock, apperrors.CodeInvalidStatusTransition, apperrors.CodeDuplicateWebhookEvent, apperrors.CodeIdempotencyConflict:
		return 409
	case apperrors.CodeNotImplemented:
		return 501
	default:
		return 500
	}
}
