package domain

import (
	"errors"
	"testing"
)

func TestValidationError_Error(t *testing.T) {
	err := NewValidationError("email", "invalid format")
	expected := "validation error on field 'email': invalid format"
	if err.Error() != expected {
		t.Errorf("ValidationError.Error() = %v, want %v", err.Error(), expected)
	}
}

func TestValidationErrors_Add(t *testing.T) {
	errs := NewValidationErrors()
	errs.Add("field1", "error1")
	errs.Add("field2", "error2")

	if len(errs.Errors) != 2 {
		t.Errorf("ValidationErrors.Errors length = %d, want 2", len(errs.Errors))
	}
}

func TestValidationErrors_HasErrors(t *testing.T) {
	tests := []struct {
		name string
		errs *ValidationErrors
		want bool
	}{
		{"empty", NewValidationErrors(), false},
		{"with errors", func() *ValidationErrors {
			e := NewValidationErrors()
			e.Add("field", "error")
			return e
		}(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errs.HasErrors(); got != tt.want {
				t.Errorf("ValidationErrors.HasErrors() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidationErrors_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors int
		want   string
	}{
		{"no errors", 0, "validation failed"},
		{"one error", 1, "validation error on field 'field0': error0"},
		{"multiple errors", 3, "validation failed: 3 errors"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NewValidationErrors()
			for i := 0; i < tt.errors; i++ {
				errs.Add("field"+intToString(int64(i)), "error"+intToString(int64(i)))
			}
			if got := errs.Error(); got != tt.want {
				t.Errorf("ValidationErrors.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("User", "123")

	if err.Error() != "User with ID '123' not found" {
		t.Errorf("NotFoundError.Error() = %v", err.Error())
	}

	if !errors.Is(err, ErrNotFound) {
		t.Error("NotFoundError should match ErrNotFound with errors.Is")
	}

	if errors.Unwrap(err) != ErrNotFound {
		t.Error("NotFoundError.Unwrap() should return ErrNotFound")
	}
}

func TestNotFoundError_NoID(t *testing.T) {
	err := NewNotFoundError("User", "")
	if err.Error() != "User not found" {
		t.Errorf("NotFoundError.Error() = %v", err.Error())
	}
}

func TestAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("User", "email", "test@example.com")

	expected := "User with email 'test@example.com' already exists"
	if err.Error() != expected {
		t.Errorf("AlreadyExistsError.Error() = %v, want %v", err.Error(), expected)
	}

	if !errors.Is(err, ErrAlreadyExists) {
		t.Error("AlreadyExistsError should match ErrAlreadyExists with errors.Is")
	}
}

func TestAlreadyExistsError_NoField(t *testing.T) {
	err := NewAlreadyExistsError("User", "", "")
	if err.Error() != "User already exists" {
		t.Errorf("AlreadyExistsError.Error() = %v", err.Error())
	}
}

func TestUnauthorizedError(t *testing.T) {
	tests := []struct {
		name   string
		reason string
		want   string
	}{
		{"with reason", "invalid token", "unauthorized: invalid token"},
		{"no reason", "", "unauthorized"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewUnauthorizedError(tt.reason)
			if err.Error() != tt.want {
				t.Errorf("UnauthorizedError.Error() = %v, want %v", err.Error(), tt.want)
			}
			if !errors.Is(err, ErrUnauthorized) {
				t.Error("UnauthorizedError should match ErrUnauthorized")
			}
		})
	}
}

func TestForbiddenError(t *testing.T) {
	tests := []struct {
		name     string
		action   string
		resource string
		want     string
	}{
		{"with details", "delete", "user", "forbidden: cannot delete user"},
		{"no details", "", "", "forbidden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewForbiddenError(tt.action, tt.resource)
			if err.Error() != tt.want {
				t.Errorf("ForbiddenError.Error() = %v, want %v", err.Error(), tt.want)
			}
			if !errors.Is(err, ErrForbidden) {
				t.Error("ForbiddenError should match ErrForbidden")
			}
		})
	}
}

func TestConflictError(t *testing.T) {
	tests := []struct {
		name   string
		entity string
		reason string
		want   string
	}{
		{"with reason", "User", "email already in use", "conflict on User: email already in use"},
		{"no reason", "User", "", "conflict on User"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewConflictError(tt.entity, tt.reason)
			if err.Error() != tt.want {
				t.Errorf("ConflictError.Error() = %v, want %v", err.Error(), tt.want)
			}
			if !errors.Is(err, ErrConflict) {
				t.Error("ConflictError should match ErrConflict")
			}
		})
	}
}

func TestInternalError(t *testing.T) {
	tests := []struct {
		name    string
		message string
		cause   error
		want    string
	}{
		{"with message and cause", "database error", errors.New("connection failed"), "internal error: database error: connection failed"},
		{"message only", "unexpected error", nil, "internal error: unexpected error"},
		{"cause only", "", errors.New("some error"), "internal error: some error"},
		{"neither", "", nil, "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewInternalError(tt.message, tt.cause)
			if err.Error() != tt.want {
				t.Errorf("InternalError.Error() = %v, want %v", err.Error(), tt.want)
			}
			if !errors.Is(err, ErrInternal) {
				t.Error("InternalError should match ErrInternal")
			}
		})
	}
}

func TestInternalError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewInternalError("wrapper", cause)

	if errors.Unwrap(err) != cause {
		t.Error("InternalError.Unwrap() should return the cause")
	}

	errNoCause := NewInternalError("no cause", nil)
	if errors.Unwrap(errNoCause) != ErrInternal {
		t.Error("InternalError.Unwrap() with no cause should return ErrInternal")
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"not found error", NewNotFoundError("User", "1"), true},
		{"base error", ErrNotFound, true},
		{"other error", errors.New("other"), false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"already exists error", NewAlreadyExistsError("User", "email", "test@example.com"), true},
		{"base error", ErrAlreadyExists, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAlreadyExists(tt.err); got != tt.want {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"validation error", NewValidationError("field", "error"), true},
		{"validation errors", func() error {
			e := NewValidationErrors()
			e.Add("field", "error")
			return e
		}(), true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidation(tt.err); got != tt.want {
				t.Errorf("IsValidation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnauthorized(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unauthorized error", NewUnauthorizedError("reason"), true},
		{"base error", ErrUnauthorized, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUnauthorized(tt.err); got != tt.want {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsForbidden(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"forbidden error", NewForbiddenError("action", "resource"), true},
		{"base error", ErrForbidden, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsForbidden(tt.err); got != tt.want {
				t.Errorf("IsForbidden() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"conflict error", NewConflictError("entity", "reason"), true},
		{"base error", ErrConflict, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflict(tt.err); got != tt.want {
				t.Errorf("IsConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInternal(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"internal error", NewInternalError("message", nil), true},
		{"base error", ErrInternal, true},
		{"other error", errors.New("other"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInternal(tt.err); got != tt.want {
				t.Errorf("IsInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}
