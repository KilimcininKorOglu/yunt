package core

import (
	"encoding/json"
	"fmt"
)

// Request represents a JMAP API request (RFC 8620 §3.3).
type Request struct {
	Using       []string             `json:"using"`
	MethodCalls []Invocation         `json:"methodCalls"`
	CreatedIds  map[string]string    `json:"createdIds,omitempty"`
}

// Response represents a JMAP API response (RFC 8620 §3.4).
type Response struct {
	MethodResponses []Invocation      `json:"methodResponses"`
	CreatedIds      map[string]string `json:"createdIds,omitempty"`
	SessionState    string            `json:"sessionState"`
}

// Invocation represents a JMAP method call or response triple: [name, args, callId].
type Invocation struct {
	Name   string
	Args   json.RawMessage
	CallID string
}

// MarshalJSON serializes Invocation as a 3-element JSON array.
func (i Invocation) MarshalJSON() ([]byte, error) {
	return json.Marshal([]interface{}{i.Name, i.Args, i.CallID})
}

// UnmarshalJSON deserializes a 3-element JSON array into Invocation.
func (i *Invocation) UnmarshalJSON(b []byte) error {
	var raw [3]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return fmt.Errorf("invocation must be a 3-element array: %w", err)
	}
	if err := json.Unmarshal(raw[0], &i.Name); err != nil {
		return fmt.Errorf("invocation name must be a string: %w", err)
	}
	i.Args = raw[1]
	if err := json.Unmarshal(raw[2], &i.CallID); err != nil {
		return fmt.Errorf("invocation callId must be a string: %w", err)
	}
	return nil
}

// SessionResource represents the JMAP Session object (RFC 8620 §2).
type SessionResource struct {
	Capabilities    map[string]json.RawMessage `json:"capabilities"`
	Accounts        map[string]Account         `json:"accounts"`
	PrimaryAccounts map[string]string          `json:"primaryAccounts"`
	Username        string                     `json:"username"`
	APIUrl          string                     `json:"apiUrl"`
	DownloadUrl     string                     `json:"downloadUrl"`
	UploadUrl       string                     `json:"uploadUrl"`
	EventSourceUrl  string                     `json:"eventSourceUrl"`
	State           string                     `json:"state"`
}

// Account represents a JMAP account in the Session (RFC 8620 §2).
type Account struct {
	Name                string                     `json:"name"`
	IsPersonal          bool                       `json:"isPersonal"`
	IsReadOnly          bool                       `json:"isReadOnly"`
	AccountCapabilities map[string]json.RawMessage `json:"accountCapabilities"`
}

// CoreCapability represents urn:ietf:params:jmap:core capability (RFC 8620 §2).
type CoreCapability struct {
	MaxSizeUpload       int64    `json:"maxSizeUpload"`
	MaxConcurrentUpload int      `json:"maxConcurrentUpload"`
	MaxSizeRequest      int64    `json:"maxSizeRequest"`
	MaxConcurrentReqs   int      `json:"maxConcurrentRequests"`
	MaxCallsInRequest   int      `json:"maxCallsInRequest"`
	MaxObjectsInGet     int      `json:"maxObjectsInGet"`
	MaxObjectsInSet     int      `json:"maxObjectsInSet"`
	CollationAlgorithms []string `json:"collationAlgorithms"`
}

// MailCapability represents urn:ietf:params:jmap:mail capability (RFC 8621 §2).
type MailCapability struct {
	MaxMailboxesPerEmail   *int     `json:"maxMailboxesPerEmail"`
	MaxMailboxDepth        *int     `json:"maxMailboxDepth"`
	MaxSizeMailboxName     int      `json:"maxSizeMailboxName"`
	MaxSizeAttachmentsPerEmail int64 `json:"maxSizeAttachmentsPerEmail"`
	EmailQuerySortOptions  []string `json:"emailQuerySortOptions"`
	MayCreateTopLevelMailbox bool   `json:"mayCreateTopLevelMailbox"`
}

// SubmissionCapability represents urn:ietf:params:jmap:submission capability.
type SubmissionCapability struct {
	MaxDelayedSend       int                `json:"maxDelayedSend"`
	SubmissionExtensions map[string][]string `json:"submissionExtensions"`
}

// ContactsCapability represents urn:ietf:params:jmap:contacts capability (RFC 9610).
type ContactsCapability struct {
	MaxAddressBooksPerCard *int `json:"maxAddressBooksPerCard"`
	MayCreateAddressBook   bool `json:"mayCreateAddressBook"`
}

// ResultReference represents a reference to a previous method result (RFC 8620 §3.7).
type ResultReference struct {
	ResultOf string `json:"resultOf"`
	Name     string `json:"name"`
	Path     string `json:"path"`
}
