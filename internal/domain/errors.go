package domain

import (
	"errors"
	"fmt"
)

// Base domain errors that can be used for error comparison.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("entity not found")

	// ErrAlreadyExists is returned when attempting to create an entity that already exists.
	ErrAlreadyExists = errors.New("entity already exists")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrUnauthorized is returned when the user is not authenticated.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the user lacks permission for an action.
	ErrForbidden = errors.New("forbidden")

	// ErrConflict is returned when there is a conflict with the current state.
	ErrConflict = errors.New("conflict")

	// ErrInternal is returned for unexpected internal errors.
	ErrInternal = errors.New("internal error")
)

// ValidationError represents a validation failure with field-specific details.
type ValidationError struct {
	// Field is the name of the field that failed validation.
	Field string `json:"field"`
	// Message describes why validation failed.
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError for the specified field.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// ValidationErrors represents a collection of validation errors.
type ValidationErrors struct {
	// Errors contains all validation errors.
	Errors []*ValidationError `json:"errors"`
}

// Error implements the error interface.
func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("validation failed: %d errors", len(e.Errors))
}

// Add appends a new validation error to the collection.
func (e *ValidationErrors) Add(field, message string) {
	e.Errors = append(e.Errors, NewValidationError(field, message))
}

// HasErrors returns true if there are any validation errors.
func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// NewValidationErrors creates a new empty ValidationErrors collection.
func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		Errors: make([]*ValidationError, 0),
	}
}

// NotFoundError represents an error when an entity is not found.
type NotFoundError struct {
	// Entity is the type of entity that was not found.
	Entity string
	// ID is the identifier that was searched for.
	ID string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	if e.ID != "" {
		return fmt.Sprintf("%s with ID '%s' not found", e.Entity, e.ID)
	}
	return fmt.Sprintf("%s not found", e.Entity)
}

// Is implements errors.Is interface for error comparison.
func (e *NotFoundError) Is(target error) bool {
	return errors.Is(target, ErrNotFound)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// NewNotFoundError creates a new NotFoundError.
func NewNotFoundError(entity, id string) *NotFoundError {
	return &NotFoundError{
		Entity: entity,
		ID:     id,
	}
}

// AlreadyExistsError represents an error when an entity already exists.
type AlreadyExistsError struct {
	// Entity is the type of entity that already exists.
	Entity string
	// Field is the field that caused the conflict.
	Field string
	// Value is the conflicting value.
	Value string
}

// Error implements the error interface.
func (e *AlreadyExistsError) Error() string {
	if e.Field != "" && e.Value != "" {
		return fmt.Sprintf("%s with %s '%s' already exists", e.Entity, e.Field, e.Value)
	}
	return fmt.Sprintf("%s already exists", e.Entity)
}

// Is implements errors.Is interface for error comparison.
func (e *AlreadyExistsError) Is(target error) bool {
	return errors.Is(target, ErrAlreadyExists)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *AlreadyExistsError) Unwrap() error {
	return ErrAlreadyExists
}

// NewAlreadyExistsError creates a new AlreadyExistsError.
func NewAlreadyExistsError(entity, field, value string) *AlreadyExistsError {
	return &AlreadyExistsError{
		Entity: entity,
		Field:  field,
		Value:  value,
	}
}

// UnauthorizedError represents an authentication failure.
type UnauthorizedError struct {
	// Reason provides additional context for the failure.
	Reason string
}

// Error implements the error interface.
func (e *UnauthorizedError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("unauthorized: %s", e.Reason)
	}
	return "unauthorized"
}

// Is implements errors.Is interface for error comparison.
func (e *UnauthorizedError) Is(target error) bool {
	return errors.Is(target, ErrUnauthorized)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *UnauthorizedError) Unwrap() error {
	return ErrUnauthorized
}

// NewUnauthorizedError creates a new UnauthorizedError.
func NewUnauthorizedError(reason string) *UnauthorizedError {
	return &UnauthorizedError{
		Reason: reason,
	}
}

// ForbiddenError represents a permission failure.
type ForbiddenError struct {
	// Action is the action that was attempted.
	Action string
	// Resource is the resource the action was attempted on.
	Resource string
}

// Error implements the error interface.
func (e *ForbiddenError) Error() string {
	if e.Action != "" && e.Resource != "" {
		return fmt.Sprintf("forbidden: cannot %s %s", e.Action, e.Resource)
	}
	return "forbidden"
}

// Is implements errors.Is interface for error comparison.
func (e *ForbiddenError) Is(target error) bool {
	return errors.Is(target, ErrForbidden)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *ForbiddenError) Unwrap() error {
	return ErrForbidden
}

// NewForbiddenError creates a new ForbiddenError.
func NewForbiddenError(action, resource string) *ForbiddenError {
	return &ForbiddenError{
		Action:   action,
		Resource: resource,
	}
}

// ConflictError represents a conflict with the current state.
type ConflictError struct {
	// Entity is the type of entity involved in the conflict.
	Entity string
	// Reason provides additional context about the conflict.
	Reason string
}

// Error implements the error interface.
func (e *ConflictError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("conflict on %s: %s", e.Entity, e.Reason)
	}
	return fmt.Sprintf("conflict on %s", e.Entity)
}

// Is implements errors.Is interface for error comparison.
func (e *ConflictError) Is(target error) bool {
	return errors.Is(target, ErrConflict)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *ConflictError) Unwrap() error {
	return ErrConflict
}

// NewConflictError creates a new ConflictError.
func NewConflictError(entity, reason string) *ConflictError {
	return &ConflictError{
		Entity: entity,
		Reason: reason,
	}
}

// InternalError represents an unexpected internal error.
type InternalError struct {
	// Cause is the underlying error that caused the failure.
	Cause error
	// Message provides additional context.
	Message string
}

// Error implements the error interface.
func (e *InternalError) Error() string {
	if e.Message != "" && e.Cause != nil {
		return fmt.Sprintf("internal error: %s: %v", e.Message, e.Cause)
	}
	if e.Message != "" {
		return fmt.Sprintf("internal error: %s", e.Message)
	}
	if e.Cause != nil {
		return fmt.Sprintf("internal error: %v", e.Cause)
	}
	return "internal error"
}

// Is implements errors.Is interface for error comparison.
func (e *InternalError) Is(target error) bool {
	return errors.Is(target, ErrInternal)
}

// Unwrap returns the underlying error for errors.Unwrap.
func (e *InternalError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrInternal
}

// NewInternalError creates a new InternalError.
func NewInternalError(message string, cause error) *InternalError {
	return &InternalError{
		Message: message,
		Cause:   cause,
	}
}

// IsNotFound checks if the error is a not found error.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if the error is an already exists error.
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsValidation checks if the error is a validation error.
func IsValidation(err error) bool {
	var ve *ValidationError
	var ves *ValidationErrors
	return errors.As(err, &ve) || errors.As(err, &ves)
}

// IsUnauthorized checks if the error is an unauthorized error.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden checks if the error is a forbidden error.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsConflict checks if the error is a conflict error.
func IsConflict(err error) bool {
	return errors.Is(err, ErrConflict)
}

// IsInternal checks if the error is an internal error.
func IsInternal(err error) bool {
	return errors.Is(err, ErrInternal)
}
