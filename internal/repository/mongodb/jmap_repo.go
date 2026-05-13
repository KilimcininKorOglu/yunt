package mongodb

import (
	"context"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// JMAPRepo implements the JMAPRepository sub-aggregate for MongoDB.
type JMAPRepo struct {
	repo          *Repository
	state         *JMAPStateRepo
	identities    *JMAPIdentityRepo
	submissions   *JMAPSubmissionRepo
	vacation      *JMAPVacationRepo
	pushSubs      *JMAPPushSubRepo
	addressBooks  *JMAPAddressBookRepo
	contactCards  *JMAPContactCardRepo
}

// NewJMAPRepo creates a new JMAP repository sub-aggregate.
func NewJMAPRepo(repo *Repository) *JMAPRepo {
	return &JMAPRepo{
		repo:         repo,
		state:        &JMAPStateRepo{repo: repo},
		identities:   &JMAPIdentityRepo{repo: repo},
		submissions:  &JMAPSubmissionRepo{repo: repo},
		vacation:     &JMAPVacationRepo{repo: repo},
		pushSubs:     &JMAPPushSubRepo{repo: repo},
		addressBooks: &JMAPAddressBookRepo{repo: repo},
		contactCards: &JMAPContactCardRepo{repo: repo},
	}
}

func (j *JMAPRepo) State() repository.StateRepository            { return j.state }
func (j *JMAPRepo) Identities() repository.IdentityRepository    { return j.identities }
func (j *JMAPRepo) Submissions() repository.SubmissionRepository  { return j.submissions }
func (j *JMAPRepo) Vacation() repository.VacationRepository       { return j.vacation }
func (j *JMAPRepo) PushSubscriptions() repository.PushSubscriptionRepository { return j.pushSubs }
func (j *JMAPRepo) AddressBooks() repository.AddressBookRepository { return j.addressBooks }
func (j *JMAPRepo) ContactCards() repository.ContactCardRepository { return j.contactCards }

// JMAPStateRepo implements StateRepository for MongoDB.
type JMAPStateRepo struct{ repo *Repository }

func (s *JMAPStateRepo) CurrentState(_ context.Context, _ domain.ID, _ string) (int64, error) {
	return 0, nil
}
func (s *JMAPStateRepo) BumpState(_ context.Context, _ domain.ID, _ string, _ domain.ID, _ string) (int64, error) {
	return 0, nil
}
func (s *JMAPStateRepo) GetChanges(_ context.Context, _ domain.ID, _ string, _ int64, _ int64) (*repository.ChangesResult, error) {
	return &repository.ChangesResult{}, nil
}

// JMAPIdentityRepo implements IdentityRepository for MongoDB.
type JMAPIdentityRepo struct{ repo *Repository }

func (r *JMAPIdentityRepo) GetByID(_ context.Context, id domain.ID) (*domain.Identity, error) {
	return nil, domain.NewNotFoundError("identity", string(id))
}
func (r *JMAPIdentityRepo) List(_ context.Context, _ domain.ID) ([]*domain.Identity, error) {
	return nil, nil
}
func (r *JMAPIdentityRepo) Create(_ context.Context, _ *domain.Identity) error { return nil }
func (r *JMAPIdentityRepo) Update(_ context.Context, _ *domain.Identity) error { return nil }
func (r *JMAPIdentityRepo) Delete(_ context.Context, _ domain.ID) error        { return nil }

// JMAPSubmissionRepo implements SubmissionRepository for MongoDB.
type JMAPSubmissionRepo struct{ repo *Repository }

func (r *JMAPSubmissionRepo) GetByID(_ context.Context, id domain.ID) (*domain.EmailSubmission, error) {
	return nil, domain.NewNotFoundError("submission", string(id))
}
func (r *JMAPSubmissionRepo) List(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.EmailSubmission], error) {
	return &repository.ListResult[*domain.EmailSubmission]{}, nil
}
func (r *JMAPSubmissionRepo) Create(_ context.Context, _ *domain.EmailSubmission) error { return nil }
func (r *JMAPSubmissionRepo) Update(_ context.Context, _ *domain.EmailSubmission) error { return nil }
func (r *JMAPSubmissionRepo) Delete(_ context.Context, _ domain.ID) error               { return nil }
func (r *JMAPSubmissionRepo) GetPending(_ context.Context) ([]*domain.EmailSubmission, error) {
	return nil, nil
}

// JMAPVacationRepo implements VacationRepository for MongoDB.
type JMAPVacationRepo struct{ repo *Repository }

func (r *JMAPVacationRepo) GetByUserID(_ context.Context, userID domain.ID) (*domain.VacationResponse, error) {
	return nil, domain.NewNotFoundError("vacation", string(userID))
}
func (r *JMAPVacationRepo) Set(_ context.Context, _ *domain.VacationResponse) error { return nil }

// JMAPPushSubRepo implements PushSubscriptionRepository for MongoDB.
type JMAPPushSubRepo struct{ repo *Repository }

func (r *JMAPPushSubRepo) GetByID(_ context.Context, id domain.ID) (*domain.PushSubscription, error) {
	return nil, domain.NewNotFoundError("push_subscription", string(id))
}
func (r *JMAPPushSubRepo) ListByUser(_ context.Context, _ domain.ID) ([]*domain.PushSubscription, error) {
	return nil, nil
}
func (r *JMAPPushSubRepo) Create(_ context.Context, _ *domain.PushSubscription) error { return nil }
func (r *JMAPPushSubRepo) Update(_ context.Context, _ *domain.PushSubscription) error { return nil }
func (r *JMAPPushSubRepo) Delete(_ context.Context, _ domain.ID) error                { return nil }
func (r *JMAPPushSubRepo) DeleteExpired(_ context.Context) (int64, error)              { return 0, nil }

// JMAPAddressBookRepo implements AddressBookRepository for MongoDB.
type JMAPAddressBookRepo struct{ repo *Repository }

func (r *JMAPAddressBookRepo) GetByID(_ context.Context, id domain.ID) (*domain.AddressBook, error) {
	return nil, domain.NewNotFoundError("address_book", string(id))
}
func (r *JMAPAddressBookRepo) List(_ context.Context, _ domain.ID) ([]*domain.AddressBook, error) {
	return nil, nil
}
func (r *JMAPAddressBookRepo) Create(_ context.Context, _ *domain.AddressBook) error { return nil }
func (r *JMAPAddressBookRepo) Update(_ context.Context, _ *domain.AddressBook) error { return nil }
func (r *JMAPAddressBookRepo) Delete(_ context.Context, _ domain.ID) error           { return nil }
func (r *JMAPAddressBookRepo) GetDefault(_ context.Context, userID domain.ID) (*domain.AddressBook, error) {
	return nil, domain.NewNotFoundError("address_book", "default")
}

// JMAPContactCardRepo implements ContactCardRepository for MongoDB.
type JMAPContactCardRepo struct{ repo *Repository }

func (r *JMAPContactCardRepo) GetByID(_ context.Context, id domain.ID) (*domain.ContactCard, error) {
	return nil, domain.NewNotFoundError("contact_card", string(id))
}
func (r *JMAPContactCardRepo) GetByUID(_ context.Context, _ domain.ID, uid string) (*domain.ContactCard, error) {
	return nil, domain.NewNotFoundError("contact_card", uid)
}
func (r *JMAPContactCardRepo) List(_ context.Context, _ domain.ID, _ *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	return &repository.ListResult[*domain.ContactCard]{}, nil
}
func (r *JMAPContactCardRepo) Create(_ context.Context, _ *domain.ContactCard) error { return nil }
func (r *JMAPContactCardRepo) Update(_ context.Context, _ *domain.ContactCard) error { return nil }
func (r *JMAPContactCardRepo) Delete(_ context.Context, _ domain.ID) error           { return nil }
func (r *JMAPContactCardRepo) Query(_ context.Context, _ domain.ID, _ *domain.JMAPContactFilter, _ *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	return &repository.ListResult[*domain.ContactCard]{}, nil
}
func (r *JMAPContactCardRepo) DeleteByAddressBook(_ context.Context, _ domain.ID) (int64, error) {
	return 0, nil
}
