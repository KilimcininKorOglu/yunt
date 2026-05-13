package repository

import (
	"context"

	"yunt/internal/domain"
)

// JMAPRepository groups all JMAP-specific data access interfaces.
type JMAPRepository interface {
	State() StateRepository
	Identities() IdentityRepository
	Submissions() SubmissionRepository
	Vacation() VacationRepository
	PushSubscriptions() PushSubscriptionRepository
	AddressBooks() AddressBookRepository
	ContactCards() ContactCardRepository
}

// StateRepository manages JMAP state tracking for /changes support.
type StateRepository interface {
	CurrentState(ctx context.Context, accountID domain.ID, typeName string) (int64, error)
	BumpState(ctx context.Context, accountID domain.ID, typeName string, entityID domain.ID, changeType string) (int64, error)
	GetChanges(ctx context.Context, accountID domain.ID, typeName string, sinceState int64, maxChanges int64) (*ChangesResult, error)
}

// ChangesResult holds the result of a JMAP /changes query.
type ChangesResult struct {
	OldState  int64
	NewState  int64
	Created   []domain.ID
	Updated   []domain.ID
	Destroyed []domain.ID
	HasMore   bool
}

// IdentityRepository provides data access for JMAP Identity entities.
type IdentityRepository interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.Identity, error)
	List(ctx context.Context, userID domain.ID) ([]*domain.Identity, error)
	Create(ctx context.Context, identity *domain.Identity) error
	Update(ctx context.Context, identity *domain.Identity) error
	Delete(ctx context.Context, id domain.ID) error
}

// SubmissionRepository provides data access for JMAP EmailSubmission entities.
type SubmissionRepository interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.EmailSubmission, error)
	List(ctx context.Context, userID domain.ID, opts *ListOptions) (*ListResult[*domain.EmailSubmission], error)
	Create(ctx context.Context, submission *domain.EmailSubmission) error
	Update(ctx context.Context, submission *domain.EmailSubmission) error
	Delete(ctx context.Context, id domain.ID) error
	GetPending(ctx context.Context) ([]*domain.EmailSubmission, error)
}

// VacationRepository provides data access for JMAP VacationResponse.
type VacationRepository interface {
	GetByUserID(ctx context.Context, userID domain.ID) (*domain.VacationResponse, error)
	Set(ctx context.Context, vacation *domain.VacationResponse) error
}

// PushSubscriptionRepository provides data access for JMAP PushSubscription entities.
type PushSubscriptionRepository interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.PushSubscription, error)
	ListByUser(ctx context.Context, userID domain.ID) ([]*domain.PushSubscription, error)
	Create(ctx context.Context, sub *domain.PushSubscription) error
	Update(ctx context.Context, sub *domain.PushSubscription) error
	Delete(ctx context.Context, id domain.ID) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// AddressBookRepository provides data access for JMAP AddressBook entities.
type AddressBookRepository interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.AddressBook, error)
	List(ctx context.Context, userID domain.ID) ([]*domain.AddressBook, error)
	Create(ctx context.Context, book *domain.AddressBook) error
	Update(ctx context.Context, book *domain.AddressBook) error
	Delete(ctx context.Context, id domain.ID) error
	GetDefault(ctx context.Context, userID domain.ID) (*domain.AddressBook, error)
}

// ContactCardRepository provides data access for JMAP ContactCard entities.
type ContactCardRepository interface {
	GetByID(ctx context.Context, id domain.ID) (*domain.ContactCard, error)
	GetByUID(ctx context.Context, userID domain.ID, uid string) (*domain.ContactCard, error)
	List(ctx context.Context, userID domain.ID, opts *ListOptions) (*ListResult[*domain.ContactCard], error)
	Create(ctx context.Context, card *domain.ContactCard) error
	Update(ctx context.Context, card *domain.ContactCard) error
	Delete(ctx context.Context, id domain.ID) error
	Query(ctx context.Context, userID domain.ID, filter *domain.JMAPContactFilter, opts *ListOptions) (*ListResult[*domain.ContactCard], error)
	DeleteByAddressBook(ctx context.Context, bookID domain.ID) (int64, error)
}
