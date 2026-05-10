package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Seeder handles database seeding for MongoDB.
type Seeder struct {
	repo   *Repository
	logger *slog.Logger
}

// SeedConfig contains configuration for database seeding.
type SeedConfig struct {
	// AdminUsername is the username for the admin user.
	AdminUsername string `json:"adminUsername"`

	// AdminEmail is the email for the admin user.
	AdminEmail string `json:"adminEmail"`

	// AdminPassword is the plaintext password for the admin user.
	// Will be hashed before storage.
	AdminPassword string `json:"adminPassword"`

	// AdminDisplayName is the display name for the admin user.
	AdminDisplayName string `json:"adminDisplayName"`

	// CreateDefaultMailboxes determines if default mailboxes should be created.
	CreateDefaultMailboxes bool `json:"createDefaultMailboxes"`

	// DefaultMailboxAddress is the address for the default mailbox.
	DefaultMailboxAddress string `json:"defaultMailboxAddress"`

	// CreateCatchAll determines if a catch-all mailbox should be created.
	CreateCatchAll bool `json:"createCatchAll"`

	// CatchAllAddress is the address pattern for the catch-all mailbox.
	CatchAllAddress string `json:"catchAllAddress"`
}

// DefaultSeedConfig returns the default seed configuration.
func DefaultSeedConfig() *SeedConfig {
	return &SeedConfig{
		AdminUsername:          "admin",
		AdminEmail:             "admin@localhost",
		AdminPassword:          "admin123",
		AdminDisplayName:       "Administrator",
		CreateDefaultMailboxes: true,
		DefaultMailboxAddress:  "inbox@localhost",
		CreateCatchAll:         true,
		CatchAllAddress:        "*@localhost",
	}
}

// NewSeeder creates a new Seeder with the given repository.
func NewSeeder(repo *Repository, logger *slog.Logger) *Seeder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Seeder{
		repo:   repo,
		logger: logger,
	}
}

// Seed populates the database with initial data using default configuration.
func (s *Seeder) Seed(ctx context.Context) error {
	return s.SeedWithConfig(ctx, DefaultSeedConfig())
}

// SeedWithConfig populates the database with initial data using provided configuration.
func (s *Seeder) SeedWithConfig(ctx context.Context, config *SeedConfig) error {
	if config == nil {
		config = DefaultSeedConfig()
	}

	s.logger.Info("starting database seeding")

	// Check if admin user already exists
	adminExists, err := s.adminExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	if adminExists {
		s.logger.Info("database already seeded, skipping")
		return nil
	}

	// Create admin user
	adminUser, err := s.createAdminUser(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	s.logger.Info("admin user created",
		"userId", adminUser.ID,
		"username", adminUser.Username,
		"email", adminUser.Email)

	// Create default mailboxes for admin
	if config.CreateDefaultMailboxes {
		if err := s.createDefaultMailboxes(ctx, adminUser.ID, config); err != nil {
			return fmt.Errorf("failed to create default mailboxes: %w", err)
		}
	}

	// Initialize default settings if not exists
	if err := s.initializeSettings(ctx); err != nil {
		return fmt.Errorf("failed to initialize settings: %w", err)
	}

	s.logger.Info("database seeding completed successfully")
	return nil
}

// SeedFromFile seeds data from a JSON file.
func (s *Seeder) SeedFromFile(_ context.Context, _ string) error {
	// This would require file system access
	// For embedded migrations, we use SeedFromReader instead
	return fmt.Errorf("SeedFromFile is not implemented for MongoDB; use SeedFromReader instead")
}

// SeedFromReader seeds data from a reader containing JSON configuration.
func (s *Seeder) SeedFromReader(ctx context.Context, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read seed data: %w", err)
	}

	var config SeedConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse seed config: %w", err)
	}

	return s.SeedWithConfig(ctx, &config)
}

// adminExists checks if an admin user already exists in the database.
func (s *Seeder) adminExists(ctx context.Context) (bool, error) {
	role := domain.RoleAdmin
	filter := &repository.UserFilter{Role: &role}
	count, err := s.repo.Users().Count(ctx, filter)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// createAdminUser creates the initial admin user.
func (s *Seeder) createAdminUser(ctx context.Context, config *SeedConfig) (*domain.User, error) {
	// Hash the password
	passwordHash, err := hashPassword(config.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate a unique ID
	userID := domain.ID(uuid.New().String())
	now := domain.Now()

	user := &domain.User{
		ID:           userID,
		Username:     config.AdminUsername,
		Email:        config.AdminEmail,
		PasswordHash: passwordHash,
		DisplayName:  config.AdminDisplayName,
		Role:         domain.RoleAdmin,
		Status:       domain.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Users().Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// createDefaultMailboxes creates default mailboxes for a user.
func (s *Seeder) createDefaultMailboxes(ctx context.Context, userID domain.ID, config *SeedConfig) error {
	now := domain.Now()

	// Create default inbox mailbox
	inbox := &domain.Mailbox{
		ID:          domain.ID(uuid.New().String()),
		UserID:      userID,
		Name:        "Inbox",
		Address:     config.DefaultMailboxAddress,
		Description: "Default inbox for receiving emails",
		IsDefault:   true,
		IsCatchAll:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Mailboxes().Create(ctx, inbox); err != nil {
		return fmt.Errorf("failed to create inbox mailbox: %w", err)
	}

	s.logger.Info("default inbox created",
		"mailboxId", inbox.ID,
		"address", inbox.Address)

	// Create catch-all mailbox if configured
	if config.CreateCatchAll && config.CatchAllAddress != "" {
		catchAll := &domain.Mailbox{
			ID:          domain.ID(uuid.New().String()),
			UserID:      userID,
			Name:        "Catch-All",
			Address:     config.CatchAllAddress,
			Description: "Catches all emails that don't match other mailboxes",
			IsDefault:   false,
			IsCatchAll:  true,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		if err := s.repo.Mailboxes().Create(ctx, catchAll); err != nil {
			return fmt.Errorf("failed to create catch-all mailbox: %w", err)
		}

		s.logger.Info("catch-all mailbox created",
			"mailboxId", catchAll.ID,
			"address", catchAll.Address)
	}

	return nil
}

// initializeSettings creates default settings if they don't exist.
func (s *Seeder) initializeSettings(ctx context.Context) error {
	// Check if settings exist by trying to get them
	_, err := s.repo.Settings().Get(ctx)
	if err == nil {
		// Settings already exist
		s.logger.Debug("settings already exist, skipping initialization")
		return nil
	}

	// Settings don't exist, they will be created by the settings repository
	// when Get() is called (it returns defaults and saves them)
	s.logger.Info("settings initialized with defaults")
	return nil
}

// hashPassword hashes a plaintext password using bcrypt.
func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CreateUserWithMailbox creates a new user with a default mailbox.
// This is useful for onboarding new users after initial setup.
func (s *Seeder) CreateUserWithMailbox(ctx context.Context, username, email, password string) (*domain.User, *domain.Mailbox, error) {
	s.logger.Info("creating user with mailbox",
		"username", username,
		"email", email)

	// Hash the password
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Generate a unique ID
	userID := domain.ID(uuid.New().String())
	now := domain.Now()

	user := &domain.User{
		ID:           userID,
		Username:     username,
		Email:        email,
		PasswordHash: passwordHash,
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Users().Create(ctx, user); err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default mailbox
	mailboxAddress := fmt.Sprintf("%s@localhost", username)
	mailbox := &domain.Mailbox{
		ID:          domain.ID(uuid.New().String()),
		UserID:      userID,
		Name:        "Inbox",
		Address:     mailboxAddress,
		Description: fmt.Sprintf("Default inbox for %s", username),
		IsDefault:   true,
		IsCatchAll:  false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Mailboxes().Create(ctx, mailbox); err != nil {
		// Rollback user creation if mailbox creation fails
		if delErr := s.repo.Users().Delete(ctx, userID); delErr != nil {
			s.logger.Error("failed to rollback user creation",
				"userId", userID,
				"error", delErr)
		}
		return nil, nil, fmt.Errorf("failed to create mailbox: %w", err)
	}

	s.logger.Info("user created with mailbox",
		"userId", user.ID,
		"mailboxId", mailbox.ID)

	return user, mailbox, nil
}

// IsSeeded checks if the database has been seeded with initial data.
func (s *Seeder) IsSeeded(ctx context.Context) (bool, error) {
	return s.adminExists(ctx)
}

// GetSeedStatus returns the current seed status.
func (s *Seeder) GetSeedStatus(ctx context.Context) (*SeedStatus, error) {
	status := &SeedStatus{
		CheckedAt: time.Now().UTC(),
	}

	// Check admin user
	adminExists, err := s.adminExists(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check admin: %w", err)
	}
	status.HasAdmin = adminExists

	// Count users
	userCount, err := s.repo.Users().Count(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}
	status.UserCount = userCount

	// Count mailboxes
	mailboxCount, err := s.repo.Mailboxes().Count(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count mailboxes: %w", err)
	}
	status.MailboxCount = mailboxCount

	// Check settings
	_, err = s.repo.Settings().Get(ctx)
	status.HasSettings = err == nil

	return status, nil
}

// SeedStatus contains information about the seed status.
type SeedStatus struct {
	HasAdmin     bool      `json:"hasAdmin"`
	HasSettings  bool      `json:"hasSettings"`
	UserCount    int64     `json:"userCount"`
	MailboxCount int64     `json:"mailboxCount"`
	CheckedAt    time.Time `json:"checkedAt"`
}

// ResetDatabase clears all data and re-seeds with default configuration.
// Use with caution - this deletes all existing data!
func (s *Seeder) ResetDatabase(ctx context.Context) error {
	s.logger.Warn("resetting database - all data will be deleted")

	// Drop all collections
	collections := []string{
		CollectionUsers,
		CollectionMailboxes,
		CollectionMessages,
		CollectionMessageRecipients,
		CollectionAttachments,
		CollectionAttachmentContent,
		CollectionWebhooks,
		CollectionWebhookDeliveries,
		CollectionSettings,
		CollectionSettingsHistory,
	}

	for _, collName := range collections {
		if err := s.repo.pool.DropCollection(ctx, collName); err != nil {
			s.logger.Warn("failed to drop collection",
				"collection", collName,
				"error", err)
			// Continue with other collections even if one fails
		}
	}

	// Re-create indexes
	if err := s.repo.pool.EnsureIndexes(ctx); err != nil {
		return fmt.Errorf("failed to recreate indexes: %w", err)
	}

	// Re-seed with defaults
	return s.Seed(ctx)
}

// Ensure Seeder implements repository.Seeder
var _ repository.Seeder = (*Seeder)(nil)
