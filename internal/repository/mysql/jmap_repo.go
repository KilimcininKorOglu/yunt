package mysql

import (
	"context"
	"fmt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// JMAPRepo implements the JMAPRepository sub-aggregate for SQLite.
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

// JMAPStateRepo implements StateRepository for SQLite.
type JMAPStateRepo struct{ repo *Repository }

func (s *JMAPStateRepo) CurrentState(ctx context.Context, accountID domain.ID, typeName string) (int64, error) {
	var val int64
	err := s.repo.db().GetContext(ctx, &val,
		`SELECT COALESCE((SELECT state_value FROM jmap_state WHERE account_id = ? AND type_name = ?), 0)`,
		string(accountID), typeName)
	return val, err
}

func (s *JMAPStateRepo) BumpState(ctx context.Context, accountID domain.ID, typeName string, entityID domain.ID, changeType string) (int64, error) {
	_, err := s.repo.db().ExecContext(ctx,
		`INSERT INTO jmap_state (account_id, type_name, state_value, updated_at) VALUES (?, ?, 1, CURRENT_TIMESTAMP)
		 ON CONFLICT(account_id, type_name) DO UPDATE SET state_value = state_value + 1, updated_at = CURRENT_TIMESTAMP`,
		string(accountID), typeName)
	if err != nil {
		return 0, fmt.Errorf("failed to bump state: %w", err)
	}
	newState, err := s.CurrentState(ctx, accountID, typeName)
	if err != nil {
		return 0, err
	}
	_, err = s.repo.db().ExecContext(ctx,
		`INSERT INTO jmap_changes (account_id, type_name, state_value, entity_id, change_type) VALUES (?, ?, ?, ?, ?)`,
		string(accountID), typeName, newState, string(entityID), changeType)
	if err != nil {
		return 0, fmt.Errorf("failed to record change: %w", err)
	}
	return newState, nil
}

func (s *JMAPStateRepo) GetChanges(ctx context.Context, accountID domain.ID, typeName string, sinceState int64, maxChanges int64) (*repository.ChangesResult, error) {
	result := &repository.ChangesResult{OldState: sinceState}
	currentState, err := s.CurrentState(ctx, accountID, typeName)
	if err != nil {
		return nil, err
	}
	result.NewState = currentState

	type changeRow struct {
		EntityID   string `db:"entity_id"`
		ChangeType string `db:"change_type"`
	}
	var rows []changeRow
	err = s.repo.db().SelectContext(ctx, &rows,
		`SELECT entity_id, change_type FROM jmap_changes
		 WHERE account_id = ? AND type_name = ? AND state_value > ?
		 ORDER BY state_value ASC LIMIT ?`,
		string(accountID), typeName, sinceState, maxChanges+1)
	if err != nil {
		return nil, fmt.Errorf("failed to get changes: %w", err)
	}

	if int64(len(rows)) > maxChanges {
		result.HasMore = true
		rows = rows[:maxChanges]
	}

	created := make(map[domain.ID]bool)
	updated := make(map[domain.ID]bool)
	destroyed := make(map[domain.ID]bool)

	for _, row := range rows {
		id := domain.ID(row.EntityID)
		switch row.ChangeType {
		case "created":
			created[id] = true
		case "updated":
			if !created[id] {
				updated[id] = true
			}
		case "destroyed":
			if created[id] {
				delete(created, id)
			} else {
				delete(updated, id)
				destroyed[id] = true
			}
		}
	}

	for id := range created {
		result.Created = append(result.Created, id)
	}
	for id := range updated {
		result.Updated = append(result.Updated, id)
	}
	for id := range destroyed {
		result.Destroyed = append(result.Destroyed, id)
	}

	return result, nil
}

// JMAPIdentityRepo implements IdentityRepository for SQLite.
type JMAPIdentityRepo struct{ repo *Repository }

func (r *JMAPIdentityRepo) GetByID(ctx context.Context, id domain.ID) (*domain.Identity, error) {
	return nil, domain.NewNotFoundError("identity", string(id))
}
func (r *JMAPIdentityRepo) List(ctx context.Context, userID domain.ID) ([]*domain.Identity, error) {
	return nil, nil
}
func (r *JMAPIdentityRepo) Create(ctx context.Context, identity *domain.Identity) error { return nil }
func (r *JMAPIdentityRepo) Update(ctx context.Context, identity *domain.Identity) error { return nil }
func (r *JMAPIdentityRepo) Delete(ctx context.Context, id domain.ID) error              { return nil }

// JMAPSubmissionRepo implements SubmissionRepository for SQLite.
type JMAPSubmissionRepo struct{ repo *Repository }

func (r *JMAPSubmissionRepo) GetByID(ctx context.Context, id domain.ID) (*domain.EmailSubmission, error) {
	return nil, domain.NewNotFoundError("submission", string(id))
}
func (r *JMAPSubmissionRepo) List(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.EmailSubmission], error) {
	return &repository.ListResult[*domain.EmailSubmission]{}, nil
}
func (r *JMAPSubmissionRepo) Create(ctx context.Context, submission *domain.EmailSubmission) error {
	return nil
}
func (r *JMAPSubmissionRepo) Update(ctx context.Context, submission *domain.EmailSubmission) error {
	return nil
}
func (r *JMAPSubmissionRepo) Delete(ctx context.Context, id domain.ID) error { return nil }
func (r *JMAPSubmissionRepo) GetPending(ctx context.Context) ([]*domain.EmailSubmission, error) {
	return nil, nil
}

// JMAPVacationRepo implements VacationRepository for SQLite.
type JMAPVacationRepo struct{ repo *Repository }

func (r *JMAPVacationRepo) GetByUserID(ctx context.Context, userID domain.ID) (*domain.VacationResponse, error) {
	return nil, domain.NewNotFoundError("vacation", string(userID))
}
func (r *JMAPVacationRepo) Set(ctx context.Context, vacation *domain.VacationResponse) error {
	return nil
}

// JMAPPushSubRepo implements PushSubscriptionRepository for SQLite.
type JMAPPushSubRepo struct{ repo *Repository }

func (r *JMAPPushSubRepo) GetByID(ctx context.Context, id domain.ID) (*domain.PushSubscription, error) {
	return nil, domain.NewNotFoundError("push_subscription", string(id))
}
func (r *JMAPPushSubRepo) ListByUser(ctx context.Context, userID domain.ID) ([]*domain.PushSubscription, error) {
	return nil, nil
}
func (r *JMAPPushSubRepo) Create(ctx context.Context, sub *domain.PushSubscription) error { return nil }
func (r *JMAPPushSubRepo) Update(ctx context.Context, sub *domain.PushSubscription) error { return nil }
func (r *JMAPPushSubRepo) Delete(ctx context.Context, id domain.ID) error                 { return nil }
func (r *JMAPPushSubRepo) DeleteExpired(ctx context.Context) (int64, error)                { return 0, nil }

// JMAPAddressBookRepo implements AddressBookRepository for SQLite.
type JMAPAddressBookRepo struct{ repo *Repository }

func (r *JMAPAddressBookRepo) GetByID(ctx context.Context, id domain.ID) (*domain.AddressBook, error) {
	return nil, domain.NewNotFoundError("address_book", string(id))
}
func (r *JMAPAddressBookRepo) List(ctx context.Context, userID domain.ID) ([]*domain.AddressBook, error) {
	return nil, nil
}
func (r *JMAPAddressBookRepo) Create(ctx context.Context, book *domain.AddressBook) error { return nil }
func (r *JMAPAddressBookRepo) Update(ctx context.Context, book *domain.AddressBook) error { return nil }
func (r *JMAPAddressBookRepo) Delete(ctx context.Context, id domain.ID) error              { return nil }
func (r *JMAPAddressBookRepo) GetDefault(ctx context.Context, userID domain.ID) (*domain.AddressBook, error) {
	return nil, domain.NewNotFoundError("address_book", "default")
}

// JMAPContactCardRepo implements ContactCardRepository for SQLite.
type JMAPContactCardRepo struct{ repo *Repository }

func (r *JMAPContactCardRepo) GetByID(ctx context.Context, id domain.ID) (*domain.ContactCard, error) {
	return nil, domain.NewNotFoundError("contact_card", string(id))
}
func (r *JMAPContactCardRepo) GetByUID(ctx context.Context, userID domain.ID, uid string) (*domain.ContactCard, error) {
	return nil, domain.NewNotFoundError("contact_card", uid)
}
func (r *JMAPContactCardRepo) List(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	return &repository.ListResult[*domain.ContactCard]{}, nil
}
func (r *JMAPContactCardRepo) Create(ctx context.Context, card *domain.ContactCard) error { return nil }
func (r *JMAPContactCardRepo) Update(ctx context.Context, card *domain.ContactCard) error { return nil }
func (r *JMAPContactCardRepo) Delete(ctx context.Context, id domain.ID) error              { return nil }
func (r *JMAPContactCardRepo) Query(ctx context.Context, userID domain.ID, filter *domain.JMAPContactFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	return &repository.ListResult[*domain.ContactCard]{}, nil
}
func (r *JMAPContactCardRepo) DeleteByAddressBook(ctx context.Context, bookID domain.ID) (int64, error) {
	return 0, nil
}
