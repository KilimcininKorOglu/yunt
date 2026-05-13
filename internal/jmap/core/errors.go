package core

import "encoding/json"

// MethodError represents a JMAP method-level error (RFC 8620 §3.6.2).
type MethodError struct {
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// NewMethodError creates a new method-level error.
func NewMethodError(errType, description string) *MethodError {
	return &MethodError{Type: errType, Description: description}
}

// ToInvocation converts a MethodError to an error Invocation response.
func (e *MethodError) ToInvocation(callID string) Invocation {
	data, _ := json.Marshal(e)
	return Invocation{Name: "error", Args: data, CallID: callID}
}

// Standard method-level error types (RFC 8620 §3.6.2).
const (
	ErrorUnknownMethod          = "unknownMethod"
	ErrorInvalidArguments       = "invalidArguments"
	ErrorForbidden              = "forbidden"
	ErrorAccountNotFound        = "accountNotFound"
	ErrorAccountNotSupported    = "accountNotSupportedByMethod"
	ErrorAccountReadOnly        = "accountReadOnly"
	ErrorRequestTooLarge        = "requestTooLarge"
	ErrorStateMismatch          = "stateMismatch"
	ErrorServerFail             = "serverFail"
	ErrorInvalidResultReference = "invalidResultReference"
)

// Standard set-level error types (RFC 8620 §5.3).
const (
	SetErrorInvalidProperties = "invalidProperties"
	SetErrorNotFound          = "notFound"
	SetErrorForbidden         = "forbidden"
	SetErrorOverQuota         = "overQuota"
	SetErrorTooLarge          = "tooLarge"
	SetErrorRateLimit         = "rateLimit"
)

// Mail-specific set error types (RFC 8621).
const (
	SetErrorMailboxHasEmail = "mailboxHasEmail"
	SetErrorMailboxHasChild = "mailboxHasChild"
	SetErrorTooManyKeywords  = "tooManyKeywords"
	SetErrorTooManyMailboxes = "tooManyMailboxes"
)

// RequestError represents a request-level error returned as HTTP 4xx with RFC 7807 body.
type RequestError struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// Request-level error type URIs (RFC 8620 §3.6.1).
const (
	RequestErrorUnknownCapability = "urn:ietf:params:jmap:error:unknownCapability"
	RequestErrorNotJSON           = "urn:ietf:params:jmap:error:notJSON"
	RequestErrorNotRequest        = "urn:ietf:params:jmap:error:notRequest"
	RequestErrorLimit             = "urn:ietf:params:jmap:error:limit"
)
