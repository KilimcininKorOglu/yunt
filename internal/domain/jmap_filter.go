package domain

// JMAPFilterOperator represents a compound JMAP filter (RFC 8620 §5.5).
// Supports AND, OR, NOT composition with nested conditions.
type JMAPFilterOperator struct {
	Operator   string        `json:"operator"`
	Conditions []interface{} `json:"conditions"`
}

// JMAPEmailFilter represents filter conditions for Email/query (RFC 8621 §4.4.1).
type JMAPEmailFilter struct {
	InMailbox                 *ID        `json:"inMailbox,omitempty"`
	InMailboxOtherThan        []ID       `json:"inMailboxOtherThan,omitempty"`
	Before                    *Timestamp `json:"before,omitempty"`
	After                     *Timestamp `json:"after,omitempty"`
	MinSize                   *int64     `json:"minSize,omitempty"`
	MaxSize                   *int64     `json:"maxSize,omitempty"`
	AllInThreadHaveKeyword    string     `json:"allInThreadHaveKeyword,omitempty"`
	SomeInThreadHaveKeyword   string     `json:"someInThreadHaveKeyword,omitempty"`
	NoneInThreadHaveKeyword   string     `json:"noneInThreadHaveKeyword,omitempty"`
	HasKeyword                string     `json:"hasKeyword,omitempty"`
	NotKeyword                string     `json:"notKeyword,omitempty"`
	HasAttachment             *bool      `json:"hasAttachment,omitempty"`
	Text                      string     `json:"text,omitempty"`
	From                      string     `json:"from,omitempty"`
	To                        string     `json:"to,omitempty"`
	Cc                        string     `json:"cc,omitempty"`
	Bcc                       string     `json:"bcc,omitempty"`
	Subject                   string     `json:"subject,omitempty"`
	Body                      string     `json:"body,omitempty"`
	Header                    []string   `json:"header,omitempty"`
}

// JMAPMailboxFilter represents filter conditions for Mailbox/query (RFC 8621 §2.3).
type JMAPMailboxFilter struct {
	ParentID       *ID    `json:"parentId,omitempty"`
	Name           string `json:"name,omitempty"`
	Role           string `json:"role,omitempty"`
	HasAnyRole     *bool  `json:"hasAnyRole,omitempty"`
	IsSubscribed   *bool  `json:"isSubscribed,omitempty"`
}

// JMAPContactFilter represents filter conditions for ContactCard/query (RFC 9610 §3.3.1).
type JMAPContactFilter struct {
	InAddressBook *ID        `json:"inAddressBook,omitempty"`
	UID           string     `json:"uid,omitempty"`
	HasMember     string     `json:"hasMember,omitempty"`
	Kind          string     `json:"kind,omitempty"`
	CreatedBefore *Timestamp `json:"createdBefore,omitempty"`
	CreatedAfter  *Timestamp `json:"createdAfter,omitempty"`
	UpdatedBefore *Timestamp `json:"updatedBefore,omitempty"`
	UpdatedAfter  *Timestamp `json:"updatedAfter,omitempty"`
	Text          string     `json:"text,omitempty"`
	Name          string     `json:"name,omitempty"`
	Email         string     `json:"email,omitempty"`
	Phone         string     `json:"phone,omitempty"`
	OnlineService string     `json:"onlineService,omitempty"`
	Address       string     `json:"address,omitempty"`
	Note          string     `json:"note,omitempty"`
}
