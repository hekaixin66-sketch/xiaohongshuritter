package main

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"
	"strings"
)

type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Retryable  bool
	Details    any
	Cause      error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Message
	}
	if e.Message == "" {
		return e.Cause.Error()
	}
	return e.Message + ": " + e.Cause.Error()
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func newAppError(code, message string, statusCode int, retryable bool, cause error, details any) *AppError {
	if statusCode == 0 {
		statusCode = http.StatusInternalServerError
	}
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Retryable:  retryable,
		Details:    details,
		Cause:      cause,
	}
}

func asAppError(err error) (*AppError, bool) {
	if err == nil {
		return nil, false
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

func classifyError(err error) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := asAppError(err); ok {
		return appErr
	}

	switch {
	case stderrors.Is(err, context.DeadlineExceeded):
		return newAppError("OPERATION_TIMEOUT", "operation timed out", http.StatusGatewayTimeout, true, err, nil)
	case stderrors.Is(err, context.Canceled):
		return newAppError("OPERATION_CANCELED", "operation canceled", http.StatusRequestTimeout, true, err, nil)
	}

	lower := strings.ToLower(err.Error())
	switch {
	case strings.Contains(lower, "account concurrency exhausted"):
		return newAppError("ACCOUNT_CONCURRENCY_EXHAUSTED", "account concurrency exhausted", http.StatusTooManyRequests, true, err, nil)
	case strings.Contains(lower, "global concurrency exhausted"):
		return newAppError("GLOBAL_CONCURRENCY_EXHAUSTED", "global concurrency exhausted", http.StatusTooManyRequests, true, err, nil)
	case strings.Contains(lower, "account not found"), strings.Contains(lower, "default tenant"), strings.Contains(lower, "default account"):
		return newAppError("INVALID_ACCOUNT_SCOPE", "invalid tenant/account selection", http.StatusBadRequest, false, err, nil)
	case strings.Contains(lower, "video file inaccessible"), strings.Contains(lower, "invalid schedule_at"),
		strings.Contains(lower, "unsupported visibility"), strings.Contains(lower, "title length exceeds"),
		strings.Contains(lower, "image path"), strings.Contains(lower, "images are required"),
		strings.Contains(lower, "video path is required"):
		return newAppError("INVALID_INPUT", "invalid publish input", http.StatusBadRequest, false, err, nil)
	case strings.Contains(lower, "account is cooling down"):
		return newAppError("ACCOUNT_COOLDOWN", "account is cooling down", http.StatusTooManyRequests, true, err, nil)
	default:
		return newAppError("INTERNAL_ERROR", "internal server error", http.StatusInternalServerError, false, err, nil)
	}
}

func mergeErrorDetails(err error, details any) any {
	appErr := classifyError(err)
	if appErr == nil {
		return details
	}
	if details != nil {
		return details
	}
	if appErr.Details != nil {
		return appErr.Details
	}
	return map[string]any{
		"error_code":    appErr.Code,
		"error_message": appErr.Message,
		"retryable":     appErr.Retryable,
		"cause":         fmt.Sprintf("%v", err),
	}
}
