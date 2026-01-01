package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"yunt/internal/domain"
)

func setupTestContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

func TestOK(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]string{"message": "hello"}
	err := OK(c, data)
	if err != nil {
		t.Fatalf("OK() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	if resp.Meta == nil {
		t.Error("expected meta to be present")
	}
}

func TestCreated(t *testing.T) {
	c, rec := setupTestContext()

	data := map[string]string{"id": "123"}
	err := Created(c, data)
	if err != nil {
		t.Fatalf("Created() returned error: %v", err)
	}

	if rec.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestNoContent(t *testing.T) {
	c, rec := setupTestContext()

	err := NoContent(c)
	if err != nil {
		t.Fatalf("NoContent() returned error: %v", err)
	}

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if rec.Body.Len() != 0 {
		t.Error("expected empty body")
	}
}

func TestPaginated(t *testing.T) {
	c, rec := setupTestContext()

	items := []string{"item1", "item2", "item3"}
	err := Paginated(c, items, 1, 10, 25)
	if err != nil {
		t.Fatalf("Paginated() returned error: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}

	pagination, ok := data["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("expected pagination to be present")
	}

	if pagination["totalItems"].(float64) != 25 {
		t.Errorf("expected totalItems to be 25, got %v", pagination["totalItems"])
	}

	if pagination["totalPages"].(float64) != 3 {
		t.Errorf("expected totalPages to be 3, got %v", pagination["totalPages"])
	}
}

func TestBadRequest(t *testing.T) {
	c, rec := setupTestContext()

	err := BadRequest(c, "invalid input")
	if err != nil {
		t.Fatalf("BadRequest() returned error: %v", err)
	}

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("expected success to be false")
	}

	if resp.Error == nil {
		t.Fatal("expected error to be present")
	}

	if resp.Error.Code != CodeBadRequest {
		t.Errorf("expected code %s, got %s", CodeBadRequest, resp.Error.Code)
	}
}

func TestUnauthorized(t *testing.T) {
	c, rec := setupTestContext()

	err := Unauthorized(c, "")
	if err != nil {
		t.Fatalf("Unauthorized() returned error: %v", err)
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Message != "Authentication required" {
		t.Errorf("expected default message, got %s", resp.Error.Message)
	}
}

func TestForbidden(t *testing.T) {
	c, rec := setupTestContext()

	err := Forbidden(c, "")
	if err != nil {
		t.Fatalf("Forbidden() returned error: %v", err)
	}

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Message != "Access denied" {
		t.Errorf("expected default message, got %s", resp.Error.Message)
	}
}

func TestNotFound(t *testing.T) {
	c, rec := setupTestContext()

	err := NotFound(c, "user not found")
	if err != nil {
		t.Fatalf("NotFound() returned error: %v", err)
	}

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != CodeNotFound {
		t.Errorf("expected code %s, got %s", CodeNotFound, resp.Error.Code)
	}
}

func TestConflict(t *testing.T) {
	c, rec := setupTestContext()

	err := Conflict(c, "resource already exists")
	if err != nil {
		t.Fatalf("Conflict() returned error: %v", err)
	}

	if rec.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d", http.StatusConflict, rec.Code)
	}
}

func TestValidationFailed(t *testing.T) {
	c, rec := setupTestContext()

	validationErrors := []*domain.ValidationError{
		{Field: "email", Message: "invalid email format"},
		{Field: "password", Message: "too short"},
	}

	err := ValidationFailed(c, validationErrors)
	if err != nil {
		t.Fatalf("ValidationFailed() returned error: %v", err)
	}

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Code != CodeValidationFailed {
		t.Errorf("expected code %s, got %s", CodeValidationFailed, resp.Error.Code)
	}

	details, ok := resp.Error.Details.([]interface{})
	if !ok {
		t.Fatal("expected details to be a slice")
	}

	if len(details) != 2 {
		t.Errorf("expected 2 validation errors, got %d", len(details))
	}
}

func TestInternalServerError(t *testing.T) {
	c, rec := setupTestContext()

	err := InternalServerError(c, "")
	if err != nil {
		t.Fatalf("InternalServerError() returned error: %v", err)
	}

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	var resp Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Error.Message != "An unexpected error occurred" {
		t.Errorf("expected default message, got %s", resp.Error.Message)
	}
}

func TestFromError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{"nil error", nil, http.StatusInternalServerError},
		{"not found", domain.ErrNotFound, http.StatusNotFound},
		{"already exists", domain.ErrAlreadyExists, http.StatusConflict},
		{"unauthorized", domain.ErrUnauthorized, http.StatusUnauthorized},
		{"forbidden", domain.ErrForbidden, http.StatusForbidden},
		{"conflict", domain.ErrConflict, http.StatusConflict},
		{"invalid input", domain.ErrInvalidInput, http.StatusBadRequest},
		{"validation error", domain.NewValidationError("field", "invalid"), http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := setupTestContext()

			err := FromError(c, tt.err)
			if err != nil {
				t.Fatalf("FromError() returned error: %v", err)
			}

			if rec.Code != tt.expectedCode {
				t.Errorf("expected status %d, got %d", tt.expectedCode, rec.Code)
			}
		})
	}
}
