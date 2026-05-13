package domain

// Thread represents a JMAP conversation thread (RFC 8621 §3).
// A thread groups related emails based on In-Reply-To/References headers.
type Thread struct {
	ID       ID   `json:"id"`
	EmailIDs []ID `json:"emailIds"`
}

// Identity represents a JMAP sending identity (RFC 8621 §6).
type Identity struct {
	ID            ID             `json:"id"`
	UserID        ID             `json:"userId"`
	Name          string         `json:"name"`
	Email         string         `json:"email"`
	ReplyTo       []EmailAddress `json:"replyTo,omitempty"`
	Bcc           []EmailAddress `json:"bcc,omitempty"`
	TextSignature string         `json:"textSignature"`
	HTMLSignature string         `json:"htmlSignature"`
	MayDelete     bool           `json:"mayDelete"`
	CreatedAt     Timestamp      `json:"createdAt"`
	UpdatedAt     Timestamp      `json:"updatedAt"`
}

// EmailSubmission represents a JMAP email submission (RFC 8621 §7).
type EmailSubmission struct {
	ID             ID                        `json:"id"`
	IdentityID     ID                        `json:"identityId"`
	EmailID        ID                        `json:"emailId"`
	ThreadID       ID                        `json:"threadId"`
	EnvelopeFrom   string                    `json:"envelopeFrom"`
	EnvelopeTo     []string                  `json:"envelopeTo"`
	SendAt         *Timestamp                `json:"sendAt,omitempty"`
	UndoStatus     string                    `json:"undoStatus"`
	DeliveryStatus map[string]DeliveryStatus `json:"deliveryStatus,omitempty"`
	CreatedAt      Timestamp                 `json:"createdAt"`
	UpdatedAt      Timestamp                 `json:"updatedAt"`
}

// DeliveryStatus represents per-recipient delivery status.
type DeliveryStatus struct {
	SmtpReply string `json:"smtpReply"`
	Delivered string `json:"delivered"`
	Displayed string `json:"displayed"`
}

// VacationResponse represents a JMAP vacation auto-reply (RFC 8621 §8).
// There is exactly one per account with id "singleton".
type VacationResponse struct {
	ID        ID         `json:"id"`
	UserID    ID         `json:"userId"`
	IsEnabled bool       `json:"isEnabled"`
	FromDate  *Timestamp `json:"fromDate,omitempty"`
	ToDate    *Timestamp `json:"toDate,omitempty"`
	Subject   string     `json:"subject"`
	TextBody  string     `json:"textBody"`
	HTMLBody  string     `json:"htmlBody"`
	UpdatedAt Timestamp  `json:"updatedAt"`
}

// PushSubscription represents a JMAP push subscription (RFC 8620 §7).
// PushSubscriptions are user-global — they do NOT take an accountId.
type PushSubscription struct {
	ID               ID         `json:"id"`
	UserID           ID         `json:"userId"`
	DeviceClientID   string     `json:"deviceClientId"`
	URL              string     `json:"url"`
	KeysP256DH       string     `json:"keys_p256dh,omitempty"`
	KeysAuth         string     `json:"keys_auth,omitempty"`
	VerificationCode string     `json:"verificationCode,omitempty"`
	Expires          *Timestamp `json:"expires,omitempty"`
	Types            []string   `json:"types,omitempty"`
	CreatedAt        Timestamp  `json:"createdAt"`
}

// AddressBook represents a JMAP contacts address book (RFC 9610).
type AddressBook struct {
	ID           ID        `json:"id"`
	UserID       ID        `json:"userId"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	SortOrder    int       `json:"sortOrder"`
	IsDefault    bool      `json:"isDefault"`
	IsSubscribed bool      `json:"isSubscribed"`
	CreatedAt    Timestamp `json:"createdAt"`
	UpdatedAt    Timestamp `json:"updatedAt"`
}

// ContactCard represents a JMAP contact card based on JSContact (RFC 9553/9610).
// Uses hybrid storage: query-relevant fields as columns, rest as JSON blob.
type ContactCard struct {
	ID             ID              `json:"id"`
	UID            string          `json:"uid"`
	UserID         ID              `json:"userId"`
	AddressBookIDs map[ID]bool     `json:"addressBookIds"`
	Kind           string          `json:"kind"`
	FullName       string          `json:"fullName"`
	Name           *ContactName    `json:"name,omitempty"`
	Emails         []ContactEmail  `json:"emails,omitempty"`
	Phones         []ContactPhone  `json:"phones,omitempty"`
	Addresses      []ContactAddr   `json:"addresses,omitempty"`
	Notes          string          `json:"notes,omitempty"`
	Photos         []ContactPhoto  `json:"photos,omitempty"`
	ExtraData      map[string]any  `json:"extraData,omitempty"`
	CreatedAt      Timestamp       `json:"createdAt"`
	UpdatedAt      Timestamp       `json:"updatedAt"`
}

// ContactName holds structured name components (JSContact RFC 9553).
type ContactName struct {
	Given   string `json:"given,omitempty"`
	Surname string `json:"surname,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	Suffix  string `json:"suffix,omitempty"`
}

// ContactEmail holds an email address entry.
type ContactEmail struct {
	Address   string `json:"address"`
	Label     string `json:"label,omitempty"`
	IsDefault bool   `json:"isDefault,omitempty"`
}

// ContactPhone holds a phone number entry.
type ContactPhone struct {
	Number string `json:"number"`
	Label  string `json:"label,omitempty"`
}

// ContactAddr holds a postal address.
type ContactAddr struct {
	Street     string `json:"street,omitempty"`
	Locality   string `json:"locality,omitempty"`
	Region     string `json:"region,omitempty"`
	PostalCode string `json:"postalCode,omitempty"`
	Country    string `json:"country,omitempty"`
	Label      string `json:"label,omitempty"`
}

// ContactPhoto holds a photo reference.
type ContactPhoto struct {
	BlobID    string `json:"blobId,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
}
