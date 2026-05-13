package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
		`INSERT INTO jmap_state (account_id, type_name, state_value, updated_at) VALUES (?, ?, 1, NOW())
		 ON CONFLICT(account_id, type_name) DO UPDATE SET state_value = state_value + 1, updated_at = NOW()`,
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
	var row struct {
		ID            string    `db:"id"`
		UserID        string    `db:"user_id"`
		Name          string    `db:"name"`
		Email         string    `db:"email"`
		ReplyTo       string    `db:"reply_to"`
		Bcc           string    `db:"bcc"`
		TextSignature string    `db:"text_signature"`
		HTMLSignature string    `db:"html_signature"`
		MayDelete     bool      `db:"may_delete"`
		CreatedAt     time.Time `db:"created_at"`
		UpdatedAt     time.Time `db:"updated_at"`
	}
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM identities WHERE id = ?`, string(id))
	if err != nil {
		return nil, domain.NewNotFoundError("identity", string(id))
	}
	identity := &domain.Identity{
		ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), Name: row.Name, Email: row.Email,
		TextSignature: row.TextSignature, HTMLSignature: row.HTMLSignature, MayDelete: row.MayDelete,
		CreatedAt: domain.Timestamp{Time: row.CreatedAt}, UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
	}
	_ = json.Unmarshal([]byte(row.ReplyTo), &identity.ReplyTo)
	_ = json.Unmarshal([]byte(row.Bcc), &identity.Bcc)
	return identity, nil
}

func (r *JMAPIdentityRepo) List(ctx context.Context, userID domain.ID) ([]*domain.Identity, error) {
	var rows []struct {
		ID            string    `db:"id"`
		UserID        string    `db:"user_id"`
		Name          string    `db:"name"`
		Email         string    `db:"email"`
		ReplyTo       string    `db:"reply_to"`
		Bcc           string    `db:"bcc"`
		TextSignature string    `db:"text_signature"`
		HTMLSignature string    `db:"html_signature"`
		MayDelete     bool      `db:"may_delete"`
		CreatedAt     time.Time `db:"created_at"`
		UpdatedAt     time.Time `db:"updated_at"`
	}
	err := r.repo.db().SelectContext(ctx, &rows, `SELECT * FROM identities WHERE user_id = ?`, string(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.Identity, len(rows))
	for i, row := range rows {
		id := &domain.Identity{
			ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), Name: row.Name, Email: row.Email,
			TextSignature: row.TextSignature, HTMLSignature: row.HTMLSignature, MayDelete: row.MayDelete,
			CreatedAt: domain.Timestamp{Time: row.CreatedAt}, UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
		}
		_ = json.Unmarshal([]byte(row.ReplyTo), &id.ReplyTo)
		_ = json.Unmarshal([]byte(row.Bcc), &id.Bcc)
		result[i] = id
	}
	return result, nil
}

func (r *JMAPIdentityRepo) Create(ctx context.Context, identity *domain.Identity) error {
	replyTo, _ := json.Marshal(identity.ReplyTo)
	bcc, _ := json.Marshal(identity.Bcc)
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO identities (id, user_id, name, email, reply_to, bcc, text_signature, html_signature, may_delete, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(identity.ID), string(identity.UserID), identity.Name, identity.Email,
		string(replyTo), string(bcc), identity.TextSignature, identity.HTMLSignature,
		identity.MayDelete, identity.CreatedAt.Time, identity.UpdatedAt.Time)
	return err
}

func (r *JMAPIdentityRepo) Update(ctx context.Context, identity *domain.Identity) error {
	replyTo, _ := json.Marshal(identity.ReplyTo)
	bcc, _ := json.Marshal(identity.Bcc)
	_, err := r.repo.db().ExecContext(ctx,
		`UPDATE identities SET name=?, email=?, reply_to=?, bcc=?, text_signature=?, html_signature=?, may_delete=?, updated_at=? WHERE id=?`,
		identity.Name, identity.Email, string(replyTo), string(bcc),
		identity.TextSignature, identity.HTMLSignature, identity.MayDelete, time.Now().UTC(), string(identity.ID))
	return err
}

func (r *JMAPIdentityRepo) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.repo.db().ExecContext(ctx, `DELETE FROM identities WHERE id = ?`, string(id))
	return err
}

// JMAPSubmissionRepo implements SubmissionRepository for SQLite.
type JMAPSubmissionRepo struct{ repo *Repository }

func (r *JMAPSubmissionRepo) GetByID(ctx context.Context, id domain.ID) (*domain.EmailSubmission, error) {
	return nil, domain.NewNotFoundError("submission", string(id))
}
func (r *JMAPSubmissionRepo) List(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.EmailSubmission], error) {
	return &repository.ListResult[*domain.EmailSubmission]{}, nil
}
func (r *JMAPSubmissionRepo) Create(ctx context.Context, submission *domain.EmailSubmission) error {
	envelopeTo, _ := json.Marshal(submission.EnvelopeTo)
	deliveryStatus, _ := json.Marshal(submission.DeliveryStatus)
	var sendAt interface{}
	if submission.SendAt != nil {
		sendAt = submission.SendAt.Time
	}
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO email_submissions (id, identity_id, email_id, thread_id, envelope_from, envelope_to, send_at, undo_status, delivery_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(submission.ID), string(submission.IdentityID), string(submission.EmailID), string(submission.ThreadID),
		submission.EnvelopeFrom, string(envelopeTo), sendAt, submission.UndoStatus, string(deliveryStatus),
		submission.CreatedAt.Time, submission.UpdatedAt.Time)
	return err
}
func (r *JMAPSubmissionRepo) Update(ctx context.Context, submission *domain.EmailSubmission) error {
	return nil
}
func (r *JMAPSubmissionRepo) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.repo.db().ExecContext(ctx, `DELETE FROM email_submissions WHERE id = ?`, string(id))
	return err
}
func (r *JMAPSubmissionRepo) GetPending(ctx context.Context) ([]*domain.EmailSubmission, error) {
	return nil, nil
}

// JMAPVacationRepo implements VacationRepository for SQLite.
type JMAPVacationRepo struct{ repo *Repository }

func (r *JMAPVacationRepo) GetByUserID(ctx context.Context, userID domain.ID) (*domain.VacationResponse, error) {
	var row struct {
		ID        string         `db:"id"`
		UserID    string         `db:"user_id"`
		IsEnabled bool           `db:"is_enabled"`
		FromDate  sql.NullTime   `db:"from_date"`
		ToDate    sql.NullTime   `db:"to_date"`
		Subject   string         `db:"subject"`
		TextBody  string         `db:"text_body"`
		HTMLBody  string         `db:"html_body"`
		UpdatedAt time.Time      `db:"updated_at"`
	}
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM vacation_responses WHERE user_id = ?`, string(userID))
	if err != nil {
		return nil, domain.NewNotFoundError("vacation", string(userID))
	}
	v := &domain.VacationResponse{
		ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), IsEnabled: row.IsEnabled,
		Subject: row.Subject, TextBody: row.TextBody, HTMLBody: row.HTMLBody,
		UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
	}
	if row.FromDate.Valid {
		ts := domain.Timestamp{Time: row.FromDate.Time}
		v.FromDate = &ts
	}
	if row.ToDate.Valid {
		ts := domain.Timestamp{Time: row.ToDate.Time}
		v.ToDate = &ts
	}
	return v, nil
}

func (r *JMAPVacationRepo) Set(ctx context.Context, vacation *domain.VacationResponse) error {
	var fromDate, toDate interface{}
	if vacation.FromDate != nil {
		fromDate = vacation.FromDate.Time
	}
	if vacation.ToDate != nil {
		toDate = vacation.ToDate.Time
	}
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO vacation_responses (id, user_id, is_enabled, from_date, to_date, subject, text_body, html_body, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET is_enabled=?, from_date=?, to_date=?, subject=?, text_body=?, html_body=?, updated_at=?`,
		"singleton", string(vacation.UserID), vacation.IsEnabled, fromDate, toDate, vacation.Subject, vacation.TextBody, vacation.HTMLBody, time.Now().UTC(),
		vacation.IsEnabled, fromDate, toDate, vacation.Subject, vacation.TextBody, vacation.HTMLBody, time.Now().UTC())
	return err
}

// JMAPPushSubRepo implements PushSubscriptionRepository for SQLite.
type JMAPPushSubRepo struct{ repo *Repository }

func (r *JMAPPushSubRepo) GetByID(ctx context.Context, id domain.ID) (*domain.PushSubscription, error) {
	return nil, domain.NewNotFoundError("push_subscription", string(id))
}
func (r *JMAPPushSubRepo) ListByUser(ctx context.Context, userID domain.ID) ([]*domain.PushSubscription, error) {
	return nil, nil
}
func (r *JMAPPushSubRepo) Create(ctx context.Context, sub *domain.PushSubscription) error {
	types, _ := json.Marshal(sub.Types)
	var expires interface{}
	if sub.Expires != nil {
		expires = sub.Expires.Time
	}
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO push_subscriptions (id, user_id, device_client_id, url, keys_p256dh, keys_auth, verification_code, expires, types, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(sub.ID), string(sub.UserID), sub.DeviceClientID, sub.URL,
		sub.KeysP256DH, sub.KeysAuth, sub.VerificationCode, expires, string(types), sub.CreatedAt.Time)
	return err
}
func (r *JMAPPushSubRepo) Update(ctx context.Context, sub *domain.PushSubscription) error { return nil }
func (r *JMAPPushSubRepo) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.repo.db().ExecContext(ctx, `DELETE FROM push_subscriptions WHERE id = ?`, string(id))
	return err
}
func (r *JMAPPushSubRepo) DeleteExpired(ctx context.Context) (int64, error) {
	result, err := r.repo.db().ExecContext(ctx, `DELETE FROM push_subscriptions WHERE expires IS NOT NULL AND expires < NOW()`)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// JMAPAddressBookRepo implements AddressBookRepository for SQLite.
type JMAPAddressBookRepo struct{ repo *Repository }

func (r *JMAPAddressBookRepo) GetByID(ctx context.Context, id domain.ID) (*domain.AddressBook, error) {
	var row struct {
		ID          string    `db:"id"`
		UserID      string    `db:"user_id"`
		Name        string    `db:"name"`
		Description string    `db:"description"`
		SortOrder   int       `db:"sort_order"`
		IsDefault   bool      `db:"is_default"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM address_books WHERE id = ?`, string(id))
	if err != nil {
		return nil, domain.NewNotFoundError("address_book", string(id))
	}
	return &domain.AddressBook{
		ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), Name: row.Name,
		Description: row.Description, SortOrder: row.SortOrder, IsDefault: row.IsDefault,
		IsSubscribed: true, CreatedAt: domain.Timestamp{Time: row.CreatedAt}, UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
	}, nil
}

func (r *JMAPAddressBookRepo) List(ctx context.Context, userID domain.ID) ([]*domain.AddressBook, error) {
	var rows []struct {
		ID          string    `db:"id"`
		UserID      string    `db:"user_id"`
		Name        string    `db:"name"`
		Description string    `db:"description"`
		SortOrder   int       `db:"sort_order"`
		IsDefault   bool      `db:"is_default"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.repo.db().SelectContext(ctx, &rows, `SELECT * FROM address_books WHERE user_id = ? ORDER BY sort_order ASC`, string(userID))
	if err != nil {
		return nil, err
	}
	result := make([]*domain.AddressBook, len(rows))
	for i, row := range rows {
		result[i] = &domain.AddressBook{
			ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), Name: row.Name,
			Description: row.Description, SortOrder: row.SortOrder, IsDefault: row.IsDefault,
			IsSubscribed: true, CreatedAt: domain.Timestamp{Time: row.CreatedAt}, UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
		}
	}
	return result, nil
}

func (r *JMAPAddressBookRepo) Create(ctx context.Context, book *domain.AddressBook) error {
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO address_books (id, user_id, name, description, sort_order, is_default, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		string(book.ID), string(book.UserID), book.Name, book.Description,
		book.SortOrder, book.IsDefault, book.CreatedAt.Time, book.UpdatedAt.Time)
	return err
}

func (r *JMAPAddressBookRepo) Update(ctx context.Context, book *domain.AddressBook) error {
	_, err := r.repo.db().ExecContext(ctx,
		`UPDATE address_books SET name=?, description=?, sort_order=?, is_default=?, updated_at=? WHERE id=?`,
		book.Name, book.Description, book.SortOrder, book.IsDefault, time.Now().UTC(), string(book.ID))
	return err
}

func (r *JMAPAddressBookRepo) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.repo.db().ExecContext(ctx, `DELETE FROM address_books WHERE id = ?`, string(id))
	return err
}

func (r *JMAPAddressBookRepo) GetDefault(ctx context.Context, userID domain.ID) (*domain.AddressBook, error) {
	var row struct {
		ID          string    `db:"id"`
		UserID      string    `db:"user_id"`
		Name        string    `db:"name"`
		Description string    `db:"description"`
		SortOrder   int       `db:"sort_order"`
		IsDefault   bool      `db:"is_default"`
		CreatedAt   time.Time `db:"created_at"`
		UpdatedAt   time.Time `db:"updated_at"`
	}
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM address_books WHERE user_id = ? AND is_default = 1 LIMIT 1`, string(userID))
	if err != nil {
		return nil, domain.NewNotFoundError("address_book", "default")
	}
	return &domain.AddressBook{
		ID: domain.ID(row.ID), UserID: domain.ID(row.UserID), Name: row.Name,
		Description: row.Description, SortOrder: row.SortOrder, IsDefault: row.IsDefault,
		IsSubscribed: true, CreatedAt: domain.Timestamp{Time: row.CreatedAt}, UpdatedAt: domain.Timestamp{Time: row.UpdatedAt},
	}, nil
}

// JMAPContactCardRepo implements ContactCardRepository for SQLite.
type JMAPContactCardRepo struct{ repo *Repository }

func (r *JMAPContactCardRepo) GetByID(ctx context.Context, id domain.ID) (*domain.ContactCard, error) {
	var row contactCardRow
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM contact_cards WHERE id = ?`, string(id))
	if err != nil {
		return nil, domain.NewNotFoundError("contact_card", string(id))
	}
	return row.toDomain(), nil
}

func (r *JMAPContactCardRepo) GetByUID(ctx context.Context, userID domain.ID, uid string) (*domain.ContactCard, error) {
	var row contactCardRow
	err := r.repo.db().GetContext(ctx, &row, `SELECT * FROM contact_cards WHERE user_id = ? AND uid = ?`, string(userID), uid)
	if err != nil {
		return nil, domain.NewNotFoundError("contact_card", uid)
	}
	return row.toDomain(), nil
}

func (r *JMAPContactCardRepo) List(ctx context.Context, userID domain.ID, opts *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	var rows []contactCardRow
	err := r.repo.db().SelectContext(ctx, &rows, `SELECT * FROM contact_cards WHERE user_id = ? ORDER BY full_name ASC`, string(userID))
	if err != nil {
		return nil, err
	}
	items := make([]*domain.ContactCard, len(rows))
	for i := range rows {
		items[i] = rows[i].toDomain()
	}
	return &repository.ListResult[*domain.ContactCard]{Items: items, Total: int64(len(items))}, nil
}

func (r *JMAPContactCardRepo) Create(ctx context.Context, card *domain.ContactCard) error {
	abIDs, _ := json.Marshal(card.AddressBookIDs)
	nameData, _ := json.Marshal(card.Name)
	emails, _ := json.Marshal(card.Emails)
	phones, _ := json.Marshal(card.Phones)
	addresses, _ := json.Marshal(card.Addresses)
	photos, _ := json.Marshal(card.Photos)
	extra, _ := json.Marshal(card.ExtraData)
	_, err := r.repo.db().ExecContext(ctx,
		`INSERT INTO contact_cards (id, uid, user_id, address_book_ids, kind, full_name, name_data, emails, phones, addresses, notes, photos, extra_data, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(card.ID), card.UID, string(card.UserID), string(abIDs), card.Kind, card.FullName,
		string(nameData), string(emails), string(phones), string(addresses), card.Notes, string(photos), string(extra),
		card.CreatedAt.Time, card.UpdatedAt.Time)
	return err
}

func (r *JMAPContactCardRepo) Update(ctx context.Context, card *domain.ContactCard) error {
	abIDs, _ := json.Marshal(card.AddressBookIDs)
	nameData, _ := json.Marshal(card.Name)
	emails, _ := json.Marshal(card.Emails)
	phones, _ := json.Marshal(card.Phones)
	addresses, _ := json.Marshal(card.Addresses)
	photos, _ := json.Marshal(card.Photos)
	extra, _ := json.Marshal(card.ExtraData)
	_, err := r.repo.db().ExecContext(ctx,
		`UPDATE contact_cards SET address_book_ids=?, kind=?, full_name=?, name_data=?, emails=?, phones=?, addresses=?, notes=?, photos=?, extra_data=?, updated_at=? WHERE id=?`,
		string(abIDs), card.Kind, card.FullName, string(nameData), string(emails), string(phones),
		string(addresses), card.Notes, string(photos), string(extra), time.Now().UTC(), string(card.ID))
	return err
}

func (r *JMAPContactCardRepo) Delete(ctx context.Context, id domain.ID) error {
	_, err := r.repo.db().ExecContext(ctx, `DELETE FROM contact_cards WHERE id = ?`, string(id))
	return err
}

func (r *JMAPContactCardRepo) Query(ctx context.Context, userID domain.ID, filter *domain.JMAPContactFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.ContactCard], error) {
	sb := strings.Builder{}
	sb.WriteString(`SELECT * FROM contact_cards WHERE user_id = ?`)
	args := []interface{}{string(userID)}

	if filter != nil {
		if filter.Text != "" {
			sb.WriteString(` AND (full_name LIKE ? OR emails LIKE ? OR phones LIKE ? OR notes LIKE ?)`)
			pattern := "%" + filter.Text + "%"
			args = append(args, pattern, pattern, pattern, pattern)
		}
		if filter.Name != "" {
			sb.WriteString(` AND full_name LIKE ?`)
			args = append(args, "%"+filter.Name+"%")
		}
		if filter.Email != "" {
			sb.WriteString(` AND emails LIKE ?`)
			args = append(args, "%"+filter.Email+"%")
		}
		if filter.Phone != "" {
			sb.WriteString(` AND phones LIKE ?`)
			args = append(args, "%"+filter.Phone+"%")
		}
		if filter.Kind != "" {
			sb.WriteString(` AND kind = ?`)
			args = append(args, filter.Kind)
		}
		if filter.UID != "" {
			sb.WriteString(` AND uid = ?`)
			args = append(args, filter.UID)
		}
		if filter.InAddressBook != nil {
			sb.WriteString(` AND address_book_ids LIKE ?`)
			args = append(args, "%"+string(*filter.InAddressBook)+"%")
		}
	}
	sb.WriteString(` ORDER BY full_name ASC`)

	var rows []contactCardRow
	if err := r.repo.db().SelectContext(ctx, &rows, sb.String(), args...); err != nil {
		return nil, err
	}
	items := make([]*domain.ContactCard, len(rows))
	for i := range rows {
		items[i] = rows[i].toDomain()
	}
	return &repository.ListResult[*domain.ContactCard]{Items: items, Total: int64(len(items))}, nil
}

func (r *JMAPContactCardRepo) DeleteByAddressBook(ctx context.Context, bookID domain.ID) (int64, error) {
	result, err := r.repo.db().ExecContext(ctx, `DELETE FROM contact_cards WHERE address_book_ids LIKE ?`, "%"+string(bookID)+"%")
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type contactCardRow struct {
	ID             string    `db:"id"`
	UID            string    `db:"uid"`
	UserID         string    `db:"user_id"`
	AddressBookIDs string    `db:"address_book_ids"`
	Kind           string    `db:"kind"`
	FullName       string    `db:"full_name"`
	NameData       string    `db:"name_data"`
	Emails         string    `db:"emails"`
	Phones         string    `db:"phones"`
	Addresses      string    `db:"addresses"`
	Notes          string    `db:"notes"`
	Photos         string    `db:"photos"`
	ExtraData      string    `db:"extra_data"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

func (r *contactCardRow) toDomain() *domain.ContactCard {
	card := &domain.ContactCard{
		ID: domain.ID(r.ID), UID: r.UID, UserID: domain.ID(r.UserID),
		Kind: r.Kind, FullName: r.FullName, Notes: r.Notes,
		CreatedAt: domain.Timestamp{Time: r.CreatedAt}, UpdatedAt: domain.Timestamp{Time: r.UpdatedAt},
	}
	_ = json.Unmarshal([]byte(r.AddressBookIDs), &card.AddressBookIDs)
	_ = json.Unmarshal([]byte(r.NameData), &card.Name)
	_ = json.Unmarshal([]byte(r.Emails), &card.Emails)
	_ = json.Unmarshal([]byte(r.Phones), &card.Phones)
	_ = json.Unmarshal([]byte(r.Addresses), &card.Addresses)
	_ = json.Unmarshal([]byte(r.Photos), &card.Photos)
	_ = json.Unmarshal([]byte(r.ExtraData), &card.ExtraData)
	return card
}
