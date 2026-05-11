// Package sqlite provides SQLite-specific implementation of the repository interfaces.
package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// Seeder handles database seeding for SQLite.
type Seeder struct {
	repo *Repository
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
		CreateDefaultMailboxes: true,
		DefaultMailboxAddress:  "inbox@localhost",
		CreateCatchAll:         true,
		CatchAllAddress:        "*@localhost",
	}
}

// NewSeeder creates a new Seeder with the given repository.
func NewSeeder(repo *Repository) *Seeder {
	return &Seeder{repo: repo}
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

	// Check if admin user already exists
	adminExists, err := s.adminExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check for existing admin: %w", err)
	}

	if adminExists {
		// Database is already seeded, skip
		return nil
	}

	// Create admin user
	adminUser, err := s.createAdminUser(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

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

	return nil
}

// SeedFromFile seeds data from a JSON file.
func (s *Seeder) SeedFromFile(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open seed file: %w", err)
	}
	defer f.Close()
	return s.SeedFromReader(ctx, f)
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
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE role = 'admin' AND deleted_at IS NULL)`

	var exists bool
	if err := s.repo.pool.GetContext(ctx, &exists, query); err != nil {
		return false, err
	}

	return exists, nil
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
		DisplayName:  "Administrator",
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
		Type:        domain.MailboxTypeSystem,
		UIDNext:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Mailboxes().Create(ctx, inbox); err != nil {
		return fmt.Errorf("failed to create inbox mailbox: %w", err)
	}

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
	}

	return nil
}

// initializeSettings creates default settings if they don't exist.
func (s *Seeder) initializeSettings(ctx context.Context) error {
	// Check if settings exist
	_, err := s.repo.Settings().Get(ctx)
	if err == nil {
		// Settings already exist
		return nil
	}

	// Settings don't exist, they will be created by the settings repository
	// when Get() is called (it returns defaults and saves them)
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
		Type:        domain.MailboxTypeSystem,
		UIDNext:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Mailboxes().Create(ctx, mailbox); err != nil {
		// Rollback user creation if mailbox creation fails
		s.repo.Users().Delete(ctx, userID)
		return nil, nil, fmt.Errorf("failed to create mailbox: %w", err)
	}

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

// Ensure Seeder implements repository.Seeder
var _ repository.Seeder = (*Seeder)(nil)
