package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// DefaultSettingsID is the default ID for the singleton settings document.
const DefaultSettingsID = "default"

// SettingsRepository implements the repository.SettingsRepository interface for MongoDB.
type SettingsRepository struct {
	repo *Repository
}

// settingsDocument is the MongoDB document representation of settings.
type settingsDocument struct {
	ID        string          `bson:"_id"`
	Data      json.RawMessage `bson:"data"`
	UpdatedAt time.Time       `bson:"updatedAt"`
}

// settingsChangeDocument is the MongoDB document representation of a settings change.
type settingsChangeDocument struct {
	ID        string    `bson:"_id"`
	FieldPath string    `bson:"fieldPath"`
	OldValue  string    `bson:"oldValue,omitempty"`
	NewValue  string    `bson:"newValue,omitempty"`
	ChangedBy *string   `bson:"changedBy,omitempty"`
	Reason    string    `bson:"reason,omitempty"`
	ChangedAt time.Time `bson:"changedAt"`
}

// NewSettingsRepository creates a new MongoDB settings repository.
func NewSettingsRepository(repo *Repository) *SettingsRepository {
	return &SettingsRepository{repo: repo}
}

// collection returns the settings collection.
func (s *SettingsRepository) collection() *mongo.Collection {
	return s.repo.collection(CollectionSettings)
}

// historyCollection returns the settings history collection.
func (s *SettingsRepository) historyCollection() *mongo.Collection {
	return s.repo.collection(CollectionSettingsHistory)
}

// Get retrieves the current settings.
func (s *SettingsRepository) Get(ctx context.Context) (*domain.Settings, error) {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{"_id": DefaultSettingsID}

	var doc settingsDocument
	if err := s.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			// Return default settings if none exist
			return domain.NewSettings(domain.ID(DefaultSettingsID)), nil
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	var settings domain.Settings
	if err := json.Unmarshal(doc.Data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	settings.ID = domain.ID(doc.ID)
	settings.UpdatedAt = domain.Timestamp{Time: doc.UpdatedAt}

	return &settings, nil
}

// GetByID retrieves settings by their ID.
func (s *SettingsRepository) GetByID(ctx context.Context, id domain.ID) (*domain.Settings, error) {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}

	var doc settingsDocument
	if err := s.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("settings", string(id))
		}
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	var settings domain.Settings
	if err := json.Unmarshal(doc.Data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings: %w", err)
	}
	settings.ID = domain.ID(doc.ID)
	settings.UpdatedAt = domain.Timestamp{Time: doc.UpdatedAt}

	return &settings, nil
}

// Save saves or updates the settings.
func (s *SettingsRepository) Save(ctx context.Context, settings *domain.Settings) error {
	ctx = s.repo.getSessionContext(ctx)

	id := string(settings.ID)
	if id == "" {
		id = DefaultSettingsID
		settings.ID = domain.ID(id)
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	doc := &settingsDocument{
		ID:        id,
		Data:      data,
		UpdatedAt: time.Now().UTC(),
	}

	filter := bson.M{"_id": id}
	opts := options.Replace().SetUpsert(true)

	_, err = s.collection().ReplaceOne(ctx, filter, doc, opts)
	if err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	return nil
}

// Update applies partial updates to settings.
func (s *SettingsRepository) Update(ctx context.Context, id domain.ID, input *domain.SettingsUpdateInput) error {
	settings, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	input.Apply(settings)
	return s.Save(ctx, settings)
}

// Reset resets settings to their default values.
func (s *SettingsRepository) Reset(ctx context.Context) error {
	defaultSettings := domain.NewSettings(domain.ID(DefaultSettingsID))
	return s.Save(ctx, defaultSettings)
}

// Exists checks if settings have been saved.
func (s *SettingsRepository) Exists(ctx context.Context) (bool, error) {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{"_id": DefaultSettingsID}
	count, err := s.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check settings existence: %w", err)
	}

	return count > 0, nil
}

// GetSMTP retrieves only the SMTP settings.
func (s *SettingsRepository) GetSMTP(ctx context.Context) (*domain.SMTPSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.SMTP, nil
}

// UpdateSMTP updates only the SMTP settings.
func (s *SettingsRepository) UpdateSMTP(ctx context.Context, update *domain.SMTPSettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.SMTP)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetIMAP retrieves only the IMAP settings.
func (s *SettingsRepository) GetIMAP(ctx context.Context) (*domain.IMAPSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.IMAP, nil
}

// UpdateIMAP updates only the IMAP settings.
func (s *SettingsRepository) UpdateIMAP(ctx context.Context, update *domain.IMAPSettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.IMAP)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetWebUI retrieves only the Web UI settings.
func (s *SettingsRepository) GetWebUI(ctx context.Context) (*domain.WebUISettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.WebUI, nil
}

// UpdateWebUI updates only the Web UI settings.
func (s *SettingsRepository) UpdateWebUI(ctx context.Context, update *domain.WebUISettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.WebUI)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetStorage retrieves only the storage settings.
func (s *SettingsRepository) GetStorage(ctx context.Context) (*domain.StorageSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.Storage, nil
}

// UpdateStorage updates only the storage settings.
func (s *SettingsRepository) UpdateStorage(ctx context.Context, update *domain.StorageSettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.Storage)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetSecurity retrieves only the security settings.
func (s *SettingsRepository) GetSecurity(ctx context.Context) (*domain.SecuritySettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.Security, nil
}

// UpdateSecurity updates only the security settings.
func (s *SettingsRepository) UpdateSecurity(ctx context.Context, update *domain.SecuritySettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.Security)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetRetention retrieves only the retention settings.
func (s *SettingsRepository) GetRetention(ctx context.Context) (*domain.RetentionSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.Retention, nil
}

// UpdateRetention updates only the retention settings.
func (s *SettingsRepository) UpdateRetention(ctx context.Context, update *domain.RetentionSettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.Retention)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetNotifications retrieves only the notification settings.
func (s *SettingsRepository) GetNotifications(ctx context.Context) (*domain.NotificationSettings, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}
	return &settings.Notifications, nil
}

// UpdateNotifications updates only the notification settings.
func (s *SettingsRepository) UpdateNotifications(ctx context.Context, update *domain.NotificationSettingsUpdate) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	update.Apply(&settings.Notifications)
	settings.UpdatedAt = domain.Now()
	return s.Save(ctx, settings)
}

// GetSettingValue retrieves a specific setting value by path.
func (s *SettingsRepository) GetSettingValue(ctx context.Context, path string) (interface{}, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Convert settings to map for path-based access
	data, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	var settingsMap map[string]interface{}
	if err := json.Unmarshal(data, &settingsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal settings map: %w", err)
	}

	// Navigate the path
	parts := strings.Split(path, ".")
	current := interface{}(settingsMap)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
			if current == nil {
				return nil, nil
			}
		} else {
			return nil, nil
		}
	}

	return current, nil
}

// SetSettingValue sets a specific setting value by path.
func (s *SettingsRepository) SetSettingValue(ctx context.Context, path string, value interface{}) error {
	settings, err := s.Get(ctx)
	if err != nil {
		return err
	}

	// Convert settings to map for path-based access
	data, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	var settingsMap map[string]interface{}
	if err := json.Unmarshal(data, &settingsMap); err != nil {
		return fmt.Errorf("failed to unmarshal settings map: %w", err)
	}

	// Navigate and set the value
	parts := strings.Split(path, ".")
	current := settingsMap

	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if next, ok := current[part].(map[string]interface{}); ok {
				current = next
			} else {
				return fmt.Errorf("invalid path: %s", path)
			}
		}
	}

	// Convert back to settings
	modifiedData, err := json.Marshal(settingsMap)
	if err != nil {
		return fmt.Errorf("failed to marshal modified settings: %w", err)
	}

	var modifiedSettings domain.Settings
	if err := json.Unmarshal(modifiedData, &modifiedSettings); err != nil {
		return fmt.Errorf("failed to unmarshal modified settings: %w", err)
	}
	modifiedSettings.ID = settings.ID
	modifiedSettings.UpdatedAt = domain.Now()

	return s.Save(ctx, &modifiedSettings)
}

// GetHistory retrieves the history of settings changes.
func (s *SettingsRepository) GetHistory(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*repository.SettingsChange], error) {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{}
	findOpts := options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}})

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	total, err := s.historyCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count history: %w", err)
	}

	cursor, err := s.historyCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []settingsChangeDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode history: %w", err)
	}

	changes := make([]*repository.SettingsChange, len(docs))
	for i, doc := range docs {
		changes[i] = s.changeToDTO(&doc)
	}

	result := &repository.ListResult[*repository.SettingsChange]{
		Items: changes,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// GetHistoryByField retrieves the history of changes for a specific field.
func (s *SettingsRepository) GetHistoryByField(ctx context.Context, fieldPath string, opts *repository.ListOptions) (*repository.ListResult[*repository.SettingsChange], error) {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{"fieldPath": fieldPath}
	findOpts := options.Find().SetSort(bson.D{{Key: "changedAt", Value: -1}})

	if opts != nil && opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	total, err := s.historyCollection().CountDocuments(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count history: %w", err)
	}

	cursor, err := s.historyCollection().Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []settingsChangeDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode history: %w", err)
	}

	changes := make([]*repository.SettingsChange, len(docs))
	for i, doc := range docs {
		changes[i] = s.changeToDTO(&doc)
	}

	result := &repository.ListResult[*repository.SettingsChange]{
		Items: changes,
		Total: total,
	}

	if opts != nil && opts.Pagination != nil {
		result.Pagination = &domain.Pagination{
			Page:    opts.Pagination.Page,
			PerPage: opts.Pagination.PerPage,
			Total:   total,
		}
		result.HasMore = opts.Pagination.Page < result.Pagination.TotalPages()
	}

	return result, nil
}

// changeToDTO converts a settings change document to DTO.
func (s *SettingsRepository) changeToDTO(doc *settingsChangeDocument) *repository.SettingsChange {
	change := &repository.SettingsChange{
		ID:        domain.ID(doc.ID),
		FieldPath: doc.FieldPath,
		OldValue:  doc.OldValue,
		NewValue:  doc.NewValue,
		Reason:    doc.Reason,
		ChangedAt: domain.Timestamp{Time: doc.ChangedAt},
	}

	if doc.ChangedBy != nil {
		id := domain.ID(*doc.ChangedBy)
		change.ChangedBy = &id
	}

	return change
}

// Revert reverts settings to a specific historical version.
func (s *SettingsRepository) Revert(ctx context.Context, changeID domain.ID) error {
	ctx = s.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(changeID)}

	var doc settingsChangeDocument
	if err := s.historyCollection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return domain.NewNotFoundError("settings change", string(changeID))
		}
		return fmt.Errorf("failed to get change: %w", err)
	}

	// Set the old value back
	return s.SetSettingValue(ctx, doc.FieldPath, doc.OldValue)
}

// Export exports settings in a portable format.
func (s *SettingsRepository) Export(ctx context.Context) (*repository.SettingsExport, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal settings: %w", err)
	}

	return &repository.SettingsExport{
		Version:    "1.0",
		ExportedAt: domain.Now(),
		Settings:   settings,
		Checksum:   fmt.Sprintf("%x", len(data)), // Simple checksum
	}, nil
}

// Import imports settings from a portable format.
func (s *SettingsRepository) Import(ctx context.Context, data *repository.SettingsExport, merge bool) error {
	if data.Settings == nil {
		return fmt.Errorf("no settings data provided")
	}

	if merge {
		// Get current settings and merge
		current, err := s.Get(ctx)
		if err != nil {
			return err
		}

		// Merge imported settings into current
		// For simplicity, we'll just use the imported settings
		data.Settings.ID = current.ID
	} else {
		data.Settings.ID = domain.ID(DefaultSettingsID)
	}

	data.Settings.UpdatedAt = domain.Now()
	return s.Save(ctx, data.Settings)
}

// Validate validates the current settings.
func (s *SettingsRepository) Validate(ctx context.Context) ([]*repository.SettingsValidationError, error) {
	settings, err := s.Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := settings.Validate(); err != nil {
		if ve, ok := err.(*domain.ValidationErrors); ok {
			errors := make([]*repository.SettingsValidationError, len(ve.Errors))
			for i, e := range ve.Errors {
				errors[i] = &repository.SettingsValidationError{
					Path:     e.Field,
					Message:  e.Message,
					Severity: repository.ValidationSeverityError,
				}
			}
			return errors, nil
		}
		return []*repository.SettingsValidationError{{
			Path:     "settings",
			Message:  err.Error(),
			Severity: repository.ValidationSeverityError,
		}}, nil
	}

	return nil, nil
}

// GetDatabaseInfo retrieves information about the database connection.
func (s *SettingsRepository) GetDatabaseInfo(ctx context.Context) (*repository.DatabaseInfo, error) {
	return s.repo.DatabaseInfo(ctx)
}

// TestSMTPConnection tests the SMTP relay connection.
func (s *SettingsRepository) TestSMTPConnection(ctx context.Context) error {
	// This would require actual SMTP connection testing
	// For now, just verify settings exist
	smtpSettings, err := s.GetSMTP(ctx)
	if err != nil {
		return err
	}

	if !smtpSettings.Enabled {
		return fmt.Errorf("SMTP is not enabled")
	}

	return nil
}

// TestDatabaseConnection tests the database connection.
func (s *SettingsRepository) TestDatabaseConnection(ctx context.Context) error {
	return s.repo.Health(ctx)
}

// Ensure SettingsRepository implements repository.SettingsRepository
var _ repository.SettingsRepository = (*SettingsRepository)(nil)
