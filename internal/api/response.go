// Package api provides REST API and Web UI functionality for the Yunt mail server.
package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
)

// Response represents a standard API response structure.
type Response struct {
	// Success indicates if the request was successful.
	Success bool `json:"success"`
	// Data contains the response payload for successful requests.
	Data interface{} `json:"data,omitempty"`
	// Error contains error details for failed requests.
	Error *ErrorDetail `json:"error,omitempty"`
	// Meta contains additional metadata about the response.
	Meta *ResponseMeta `json:"meta,omitempty"`
}

// ErrorDetail contains detailed error information.
type ErrorDetail struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`
	// Message is a human-readable error message.
	Message string `json:"message"`
	// Details contains additional error details (e.g., validation errors).
	Details interface{} `json:"details,omitempty"`
}

// ResponseMeta contains metadata about the response.
type ResponseMeta struct {
	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
	// RequestID is the unique identifier for this request.
	RequestID string `json:"requestId,omitempty"`
}

// PaginatedData represents paginated response data.
type PaginatedData struct {
	// Items contains the data items.
	Items interface{} `json:"items"`
	// Pagination contains pagination information.
	Pagination *PaginationInfo `json:"pagination"`
}

// PaginationInfo contains pagination metadata.
type PaginationInfo struct {
	// Page is the current page number (1-based).
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"pageSize"`
	// TotalItems is the total number of items.
	TotalItems int64 `json:"totalItems"`
	// TotalPages is the total number of pages.
	TotalPages int `json:"totalPages"`
	// HasNext indicates if there is a next page.
	HasNext bool `json:"hasNext"`
	// HasPrev indicates if there is a previous page.
	HasPrev bool `json:"hasPrev"`
}

// Error codes for API responses.
const (
	CodeBadRequest          = "BAD_REQUEST"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeNotFound            = "NOT_FOUND"
	CodeConflict            = "CONFLICT"
	CodeValidationFailed    = "VALIDATION_FAILED"
	CodeInternalServerError = "INTERNAL_SERVER_ERROR"
	CodeServiceUnavailable  = "SERVICE_UNAVAILABLE"
)

// newResponse creates a new Response with default metadata.
func newResponse(c echo.Context) *Response {
	return &Response{
		Meta: &ResponseMeta{
			Timestamp: time.Now().UTC(),
			RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
		},
	}
}

// OK sends a successful response with the given data.
func OK(c echo.Context, data interface{}) error {
	resp := newResponse(c)
	resp.Success = true
	resp.Data = data
	return c.JSON(http.StatusOK, resp)
}

// Created sends a successful response for resource creation.
func Created(c echo.Context, data interface{}) error {
	resp := newResponse(c)
	resp.Success = true
	resp.Data = data
	return c.JSON(http.StatusCreated, resp)
}

// NoContent sends a successful response with no content.
func NoContent(c echo.Context) error {
	return c.NoContent(http.StatusNoContent)
}

// Paginated sends a successful paginated response.
func Paginated(c echo.Context, items interface{}, page, pageSize int, totalItems int64) error {
	totalPages := int(totalItems) / pageSize
	if int(totalItems)%pageSize > 0 {
		totalPages++
	}

	resp := newResponse(c)
	resp.Success = true
	resp.Data = &PaginatedData{
		Items: items,
		Pagination: &PaginationInfo{
			Page:       page,
			PageSize:   pageSize,
			TotalItems: totalItems,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}
	return c.JSON(http.StatusOK, resp)
}

// Error sends an error response with the given status code and error details.
func Error(c echo.Context, statusCode int, code, message string, details interface{}) error {
	resp := newResponse(c)
	resp.Success = false
	resp.Error = &ErrorDetail{
		Code:    code,
		Message: message,
		Details: details,
	}
	return c.JSON(statusCode, resp)
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c echo.Context, message string) error {
	return Error(c, http.StatusBadRequest, CodeBadRequest, message, nil)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c echo.Context, message string) error {
	if message == "" {
		message = "Authentication required"
	}
	return Error(c, http.StatusUnauthorized, CodeUnauthorized, message, nil)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c echo.Context, message string) error {
	if message == "" {
		message = "Access denied"
	}
	return Error(c, http.StatusForbidden, CodeForbidden, message, nil)
}

// NotFound sends a 404 Not Found response.
func NotFound(c echo.Context, message string) error {
	if message == "" {
		message = "Resource not found"
	}
	return Error(c, http.StatusNotFound, CodeNotFound, message, nil)
}

// Conflict sends a 409 Conflict response.
func Conflict(c echo.Context, message string) error {
	return Error(c, http.StatusConflict, CodeConflict, message, nil)
}

// ValidationFailed sends a 422 Unprocessable Entity response with validation errors.
func ValidationFailed(c echo.Context, errors interface{}) error {
	return Error(c, http.StatusUnprocessableEntity, CodeValidationFailed, "Validation failed", errors)
}

// InternalServerError sends a 500 Internal Server Error response.
func InternalServerError(c echo.Context, message string) error {
	if message == "" {
		message = "An unexpected error occurred"
	}
	return Error(c, http.StatusInternalServerError, CodeInternalServerError, message, nil)
}

// ServiceUnavailable sends a 503 Service Unavailable response.
func ServiceUnavailable(c echo.Context, message string) error {
	if message == "" {
		message = "Service temporarily unavailable"
	}
	return Error(c, http.StatusServiceUnavailable, CodeServiceUnavailable, message, nil)
}

// FromError converts a domain error to an appropriate HTTP response.
func FromError(c echo.Context, err error) error {
	if err == nil {
		return InternalServerError(c, "An unexpected error occurred")
	}

	// Handle domain-specific errors
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return NotFound(c, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		return Conflict(c, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		return Unauthorized(c, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		return Forbidden(c, err.Error())
	case errors.Is(err, domain.ErrConflict):
		return Conflict(c, err.Error())
	case errors.Is(err, domain.ErrInvalidInput):
		return BadRequest(c, err.Error())
	}

	// Handle validation errors
	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) {
		return ValidationFailed(c, []*domain.ValidationError{validationErr})
	}

	var validationErrs *domain.ValidationErrors
	if errors.As(err, &validationErrs) {
		return ValidationFailed(c, validationErrs.Errors)
	}

	// Default to internal server error
	return InternalServerError(c, "An unexpected error occurred")
}
