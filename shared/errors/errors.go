package errors

import (
	"fmt"
	"net/http"
)

// AppError represents an application error with HTTP status
type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"error"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// BadRequest creates a 400 error
func BadRequest(msg string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Message: msg}
}

// Unauthorized creates a 401 error
func Unauthorized(msg string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Message: msg}
}

// Forbidden creates a 403 error
func Forbidden(msg string) *AppError {
	return &AppError{Code: http.StatusForbidden, Message: msg}
}

// NotFound creates a 404 error
func NotFound(msg string) *AppError {
	return &AppError{Code: http.StatusNotFound, Message: msg}
}

// Internal creates a 500 error with underlying cause
func Internal(msg string, err error) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}

// Wrap wraps an error with context
func Wrap(err error, msg string) *AppError {
	return &AppError{Code: http.StatusInternalServerError, Message: msg, Err: err}
}
