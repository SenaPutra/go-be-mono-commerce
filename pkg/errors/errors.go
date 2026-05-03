package errors

import (
	stdErrors "errors"
	"strings"
)

type Code string

const (
	CodeValidationError         Code = "VALIDATION_ERROR"
	CodeUnauthorized            Code = "UNAUTHORIZED"
	CodeForbidden               Code = "FORBIDDEN"
	CodeNotFound                Code = "NOT_FOUND"
	CodeConflict                Code = "CONFLICT"
	CodeInsufficientStock       Code = "INSUFFICIENT_STOCK"
	CodeInvalidStatusTransition Code = "INVALID_STATUS_TRANSITION"
	CodeDuplicateWebhookEvent   Code = "DUPLICATE_WEBHOOK_EVENT"
	CodeIdempotencyConflict     Code = "IDEMPOTENCY_CONFLICT"
	CodeInternalError           Code = "INTERNAL_ERROR"
	CodeNotImplemented          Code = "NOT_IMPLEMENTED"
)

type AppError struct {
	Code    Code
	Message string
	Details interface{}
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}

func (e *AppError) Unwrap() error { return e.Err }

func New(code Code, msg string, details interface{}) *AppError {
	return &AppError{Code: code, Message: msg, Details: details}
}

func Validation(details interface{}) *AppError {
	return New(CodeValidationError, "Validation error", details)
}

func Unauthorized(details interface{}) *AppError {
	return New(CodeUnauthorized, "Unauthorized", details)
}
func Forbidden(details interface{}) *AppError { return New(CodeForbidden, "Forbidden", details) }
func NotFound(details interface{}) *AppError  { return New(CodeNotFound, "Not found", details) }
func Conflict(details interface{}) *AppError  { return New(CodeConflict, "Conflict", details) }
func Internal(err error) *AppError {
	return &AppError{Code: CodeInternalError, Message: "Internal server error", Err: err}
}

func AsAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if stdErrors.As(err, &appErr) {
		return appErr
	}
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "VALIDATION:"):
		detail := strings.Split(strings.TrimPrefix(msg, "VALIDATION:"), ";")
		return Validation(detail)
	case strings.HasPrefix(msg, "CONFLICT:"):
		return New(CodeConflict, "Conflict", []string{strings.TrimPrefix(msg, "CONFLICT:")})
	case msg == "UNAUTHORIZED" || msg == "INACTIVE":
		return New(CodeUnauthorized, "Unauthorized", nil)
	case msg == "FORBIDDEN":
		return New(CodeForbidden, "Forbidden", nil)
	case msg == "NOT_FOUND":
		return New(CodeNotFound, "Not found", nil)
	case msg == string(CodeInsufficientStock):
		return New(CodeInsufficientStock, "Conflict", nil)
	case msg == string(CodeInvalidStatusTransition):
		return New(CodeInvalidStatusTransition, "Conflict", nil)
	case msg == string(CodeDuplicateWebhookEvent):
		return New(CodeDuplicateWebhookEvent, "Conflict", nil)
	case msg == string(CodeIdempotencyConflict):
		return New(CodeIdempotencyConflict, "Conflict", nil)
	default:
		return New(CodeInternalError, "Internal server error", nil)
	}
}
