package mongodb

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// UserRepository implements the repository.UserRepository interface for MongoDB.
type UserRepository struct {
	repo *Repository
}

// userDocument is the MongoDB document representation of a user.
type userDocument struct {
	ID           string     `bson:"_id"`
	Username     string     `bson:"username"`
	Email        string     `bson:"email"`
	PasswordHash string     `bson:"passwordHash"`
	DisplayName  string     `bson:"displayName,omitempty"`
	Role         string     `bson:"role"`
	Status       string     `bson:"status"`
	AvatarURL    string     `bson:"avatarUrl,omitempty"`
	LastLoginAt  *time.Time `bson:"lastLoginAt,omitempty"`
	DeletedAt    *time.Time `bson:"deletedAt,omitempty"`
	CreatedAt    time.Time  `bson:"createdAt"`
	UpdatedAt    time.Time  `bson:"updatedAt"`
}

// NewUserRepository creates a new MongoDB user repository.
func NewUserRepository(repo *Repository) *UserRepository {
	return &UserRepository{repo: repo}
}

// collection returns the users collection.
func (u *UserRepository) collection() *mongo.Collection {
	return u.repo.collection(CollectionUsers)
}

// toDocument converts a domain.User to a MongoDB document.
func (u *UserRepository) toDocument(user *domain.User) *userDocument {
	doc := &userDocument{
		ID:           string(user.ID),
		Username:     user.Username,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		DisplayName:  user.DisplayName,
		Role:         string(user.Role),
		Status:       string(user.Status),
		AvatarURL:    user.AvatarURL,
		CreatedAt:    user.CreatedAt.Time,
		UpdatedAt:    user.UpdatedAt.Time,
	}

	if user.LastLoginAt != nil {
		t := user.LastLoginAt.Time
		doc.LastLoginAt = &t
	}

	return doc
}

// toDomain converts a MongoDB document to a domain.User.
func (u *UserRepository) toDomain(doc *userDocument) *domain.User {
	user := &domain.User{
		ID:           domain.ID(doc.ID),
		Username:     doc.Username,
		Email:        doc.Email,
		PasswordHash: doc.PasswordHash,
		DisplayName:  doc.DisplayName,
		Role:         domain.UserRole(doc.Role),
		Status:       domain.UserStatus(doc.Status),
		AvatarURL:    doc.AvatarURL,
		CreatedAt:    domain.Timestamp{Time: doc.CreatedAt},
		UpdatedAt:    domain.Timestamp{Time: doc.UpdatedAt},
	}

	if doc.LastLoginAt != nil {
		ts := domain.Timestamp{Time: *doc.LastLoginAt}
		user.LastLoginAt = &ts
	}

	return user
}

// GetByID retrieves a user by their unique identifier.
func (u *UserRepository) GetByID(ctx context.Context, id domain.ID) (*domain.User, error) {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}

	var doc userDocument
	if err := u.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("user", string(id))
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return u.toDomain(&doc), nil
}

// GetByUsername retrieves a user by their username.
func (u *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	ctx = u.repo.getSessionContext(ctx)

	// Case-insensitive search using regex
	filter := bson.M{
		"username":  bson.M{"$regex": "^" + regexp.QuoteMeta(username) + "$", "$options": "i"},
		"deletedAt": bson.M{"$exists": false},
	}

	var doc userDocument
	if err := u.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("user", username)
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return u.toDomain(&doc), nil
}

// GetByEmail retrieves a user by their email address.
func (u *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	ctx = u.repo.getSessionContext(ctx)

	// Case-insensitive search using regex
	filter := bson.M{
		"email":     bson.M{"$regex": "^" + regexp.QuoteMeta(strings.ToLower(email)) + "$", "$options": "i"},
		"deletedAt": bson.M{"$exists": false},
	}

	var doc userDocument
	if err := u.collection().FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, domain.NewNotFoundError("user", email)
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return u.toDomain(&doc), nil
}

// List retrieves users with optional filtering, sorting, and pagination.
func (u *UserRepository) List(ctx context.Context, filter *repository.UserFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	ctx = u.repo.getSessionContext(ctx)

	mongoFilter := u.buildFilter(filter)
	findOpts := u.buildFindOptions(opts)

	// Get total count
	total, err := u.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	// Execute query
	cursor, err := u.collection().Find(ctx, mongoFilter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("failed to decode users: %w", err)
	}

	users := make([]*domain.User, len(docs))
	for i, doc := range docs {
		users[i] = u.toDomain(&doc)
	}

	result := &repository.ListResult[*domain.User]{
		Items: users,
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

// buildFilter builds the MongoDB filter from repository.UserFilter.
func (u *UserRepository) buildFilter(filter *repository.UserFilter) bson.M {
	f := bson.M{}

	// Default: exclude deleted users
	includeDeleted := false
	if filter != nil {
		includeDeleted = filter.IncludeDeleted
	}

	if !includeDeleted {
		f["deletedAt"] = bson.M{"$exists": false}
	}

	if filter == nil {
		return f
	}

	if len(filter.IDs) > 0 {
		ids := make([]string, len(filter.IDs))
		for i, id := range filter.IDs {
			ids[i] = string(id)
		}
		f["_id"] = bson.M{"$in": ids}
	}

	if filter.Status != nil {
		f["status"] = string(*filter.Status)
	}

	if len(filter.Statuses) > 0 {
		statuses := make([]string, len(filter.Statuses))
		for i, s := range filter.Statuses {
			statuses[i] = string(s)
		}
		f["status"] = bson.M{"$in": statuses}
	}

	if filter.Role != nil {
		f["role"] = string(*filter.Role)
	}

	if len(filter.Roles) > 0 {
		roles := make([]string, len(filter.Roles))
		for i, r := range filter.Roles {
			roles[i] = string(r)
		}
		f["role"] = bson.M{"$in": roles}
	}

	if filter.Username != "" {
		f["username"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.Username) + "$", "$options": "i"}
	}

	if filter.Email != "" {
		f["email"] = bson.M{"$regex": "^" + regexp.QuoteMeta(filter.Email) + "$", "$options": "i"}
	}

	if filter.Search != "" {
		f["$text"] = bson.M{"$search": filter.Search}
	}

	if filter.CreatedBefore != nil {
		f["createdAt"] = bson.M{"$lt": filter.CreatedBefore.Time}
	}

	if filter.CreatedAfter != nil {
		if _, exists := f["createdAt"]; exists {
			f["createdAt"].(bson.M)["$gt"] = filter.CreatedAfter.Time
		} else {
			f["createdAt"] = bson.M{"$gt": filter.CreatedAfter.Time}
		}
	}

	if filter.LastLoginBefore != nil {
		f["lastLoginAt"] = bson.M{"$lt": filter.LastLoginBefore.Time}
	}

	if filter.LastLoginAfter != nil {
		if _, exists := f["lastLoginAt"]; exists {
			f["lastLoginAt"].(bson.M)["$gt"] = filter.LastLoginAfter.Time
		} else {
			f["lastLoginAt"] = bson.M{"$gt": filter.LastLoginAfter.Time}
		}
	}

	if filter.HasNeverLoggedIn != nil && *filter.HasNeverLoggedIn {
		f["lastLoginAt"] = bson.M{"$exists": false}
	}

	return f
}

// buildFindOptions builds MongoDB find options from repository.ListOptions.
func (u *UserRepository) buildFindOptions(opts *repository.ListOptions) *options.FindOptions {
	findOpts := options.Find()

	if opts == nil {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
		return findOpts
	}

	// Apply sorting
	if opts.Sort != nil {
		sortOrder := 1
		if opts.Sort.Order == domain.SortDesc {
			sortOrder = -1
		}
		field := u.mapSortField(opts.Sort.Field)
		findOpts.SetSort(bson.D{{Key: field, Value: sortOrder}})
	} else {
		findOpts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
	}

	// Apply pagination
	if opts.Pagination != nil {
		opts.Pagination.Normalize()
		findOpts.SetSkip(int64(opts.Pagination.Offset()))
		findOpts.SetLimit(int64(opts.Pagination.Limit()))
	}

	return findOpts
}

// mapSortField maps repository sort field to MongoDB field.
func (u *UserRepository) mapSortField(field string) string {
	switch field {
	case "username":
		return "username"
	case "email":
		return "email"
	case "displayName":
		return "displayName"
	case "createdAt":
		return "createdAt"
	case "updatedAt":
		return "updatedAt"
	case "lastLoginAt":
		return "lastLoginAt"
	case "role":
		return "role"
	case "status":
		return "status"
	default:
		return "createdAt"
	}
}

// Create creates a new user.
func (u *UserRepository) Create(ctx context.Context, user *domain.User) error {
	ctx = u.repo.getSessionContext(ctx)

	// Check for existing username
	exists, err := u.ExistsByUsername(ctx, user.Username)
	if err != nil {
		return fmt.Errorf("failed to check username existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("user", "username", user.Username)
	}

	// Check for existing email
	exists, err = u.ExistsByEmail(ctx, user.Email)
	if err != nil {
		return fmt.Errorf("failed to check email existence: %w", err)
	}
	if exists {
		return domain.NewAlreadyExistsError("user", "email", user.Email)
	}

	doc := u.toDocument(user)
	_, err = u.collection().InsertOne(ctx, doc)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return domain.NewAlreadyExistsError("user", "id", string(user.ID))
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update updates an existing user.
func (u *UserRepository) Update(ctx context.Context, user *domain.User) error {
	ctx = u.repo.getSessionContext(ctx)

	// Check if user exists
	exists, err := u.Exists(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %w", err)
	}
	if !exists {
		return domain.NewNotFoundError("user", string(user.ID))
	}

	filter := bson.M{"_id": string(user.ID)}
	update := bson.M{
		"$set": bson.M{
			"username":     user.Username,
			"email":        user.Email,
			"passwordHash": user.PasswordHash,
			"displayName":  user.DisplayName,
			"role":         string(user.Role),
			"status":       string(user.Status),
			"avatarUrl":    user.AvatarURL,
			"updatedAt":    time.Now().UTC(),
		},
	}

	if user.LastLoginAt != nil {
		update["$set"].(bson.M)["lastLoginAt"] = user.LastLoginAt.Time
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(user.ID))
	}

	return nil
}

// Delete permanently removes a user by their ID.
func (u *UserRepository) Delete(ctx context.Context, id domain.ID) error {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{"_id": string(id)}
	result, err := u.collection().DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.DeletedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// SoftDelete marks a user as deleted without removing the record.
func (u *UserRepository) SoftDelete(ctx context.Context, id domain.ID) error {
	ctx = u.repo.getSessionContext(ctx)

	now := time.Now().UTC()
	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"deletedAt": now,
			"updatedAt": now,
		},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// Restore restores a soft-deleted user.
func (u *UserRepository) Restore(ctx context.Context, id domain.ID) error {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": true},
	}
	update := bson.M{
		"$unset": bson.M{"deletedAt": ""},
		"$set":   bson.M{"updatedAt": time.Now().UTC()},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// Exists checks if a user with the given ID exists.
func (u *UserRepository) Exists(ctx context.Context, id domain.ID) (bool, error) {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}

	count, err := u.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByUsername checks if a user with the given username exists.
func (u *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"username":  bson.M{"$regex": "^" + regexp.QuoteMeta(username) + "$", "$options": "i"},
		"deletedAt": bson.M{"$exists": false},
	}

	count, err := u.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByEmail checks if a user with the given email exists.
func (u *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"email":     bson.M{"$regex": "^" + regexp.QuoteMeta(strings.ToLower(email)) + "$", "$options": "i"},
		"deletedAt": bson.M{"$exists": false},
	}

	count, err := u.collection().CountDocuments(ctx, filter, options.Count().SetLimit(1))
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return count > 0, nil
}

// Count returns the total number of users matching the filter.
func (u *UserRepository) Count(ctx context.Context, filter *repository.UserFilter) (int64, error) {
	ctx = u.repo.getSessionContext(ctx)

	mongoFilter := u.buildFilter(filter)
	count, err := u.collection().CountDocuments(ctx, mongoFilter)
	if err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return count, nil
}

// CountByRole returns the count of users grouped by role.
func (u *UserRepository) CountByRole(ctx context.Context) (map[domain.UserRole]int64, error) {
	ctx = u.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"deletedAt": bson.M{"$exists": false}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$role",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := u.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to count users by role: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[domain.UserRole]int64)
	for cursor.Next(ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result[domain.UserRole(doc.ID)] = doc.Count
	}

	return result, nil
}

// CountByStatus returns the count of users grouped by status.
func (u *UserRepository) CountByStatus(ctx context.Context) (map[domain.UserStatus]int64, error) {
	ctx = u.repo.getSessionContext(ctx)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"deletedAt": bson.M{"$exists": false}}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := u.collection().Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to count users by status: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[domain.UserStatus]int64)
	for cursor.Next(ctx) {
		var doc struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result[domain.UserStatus(doc.ID)] = doc.Count
	}

	return result, nil
}

// UpdatePassword updates a user's password hash.
func (u *UserRepository) UpdatePassword(ctx context.Context, id domain.ID, passwordHash string) error {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"passwordHash": passwordHash,
			"updatedAt":    time.Now().UTC(),
		},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateLastLogin updates the user's last login timestamp.
func (u *UserRepository) UpdateLastLogin(ctx context.Context, id domain.ID) error {
	ctx = u.repo.getSessionContext(ctx)

	now := time.Now().UTC()
	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"lastLoginAt": now,
			"updatedAt":   now,
		},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateStatus updates a user's status.
func (u *UserRepository) UpdateStatus(ctx context.Context, id domain.ID, status domain.UserStatus) error {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"status":    string(status),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// UpdateRole updates a user's role.
func (u *UserRepository) UpdateRole(ctx context.Context, id domain.ID, role domain.UserRole) error {
	ctx = u.repo.getSessionContext(ctx)

	filter := bson.M{
		"_id":       string(id),
		"deletedAt": bson.M{"$exists": false},
	}
	update := bson.M{
		"$set": bson.M{
			"role":      string(role),
			"updatedAt": time.Now().UTC(),
		},
	}

	result, err := u.collection().UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update role: %w", err)
	}

	if result.MatchedCount == 0 {
		return domain.NewNotFoundError("user", string(id))
	}

	return nil
}

// GetActiveUsers retrieves all active users.
func (u *UserRepository) GetActiveUsers(ctx context.Context, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	status := domain.StatusActive
	filter := &repository.UserFilter{Status: &status}
	return u.List(ctx, filter, opts)
}

// GetAdmins retrieves all admin users.
func (u *UserRepository) GetAdmins(ctx context.Context) ([]*domain.User, error) {
	role := domain.RoleAdmin
	filter := &repository.UserFilter{Role: &role}
	result, err := u.List(ctx, filter, nil)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// Search performs a text search across user fields.
func (u *UserRepository) Search(ctx context.Context, query string, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	filter := &repository.UserFilter{Search: query}
	return u.List(ctx, filter, opts)
}

// BulkUpdateStatus updates the status of multiple users.
func (u *UserRepository) BulkUpdateStatus(ctx context.Context, ids []domain.ID, status domain.UserStatus) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := u.UpdateStatus(ctx, id, status); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// BulkDelete permanently removes multiple users.
func (u *UserRepository) BulkDelete(ctx context.Context, ids []domain.ID) (*repository.BulkOperation, error) {
	result := repository.NewBulkOperation()

	for _, id := range ids {
		if err := u.Delete(ctx, id); err != nil {
			result.AddFailure(string(id), err)
		} else {
			result.AddSuccess()
		}
	}

	return result, nil
}

// GetUsersCreatedBetween retrieves users created within the date range.
func (u *UserRepository) GetUsersCreatedBetween(ctx context.Context, dateRange *repository.DateRangeFilter, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	filter := &repository.UserFilter{}
	if dateRange != nil {
		filter.CreatedAfter = dateRange.From
		filter.CreatedBefore = dateRange.To
	}
	return u.List(ctx, filter, opts)
}

// GetUsersWithRecentLogin retrieves users who logged in within the specified days.
func (u *UserRepository) GetUsersWithRecentLogin(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	ts := domain.Timestamp{Time: since}
	filter := &repository.UserFilter{LastLoginAfter: &ts}
	return u.List(ctx, filter, opts)
}

// GetInactiveUsers retrieves users who haven't logged in for the specified days.
func (u *UserRepository) GetInactiveUsers(ctx context.Context, days int, opts *repository.ListOptions) (*repository.ListResult[*domain.User], error) {
	before := time.Now().UTC().AddDate(0, 0, -days)
	ts := domain.Timestamp{Time: before}
	filter := &repository.UserFilter{LastLoginBefore: &ts}
	return u.List(ctx, filter, opts)
}

// Ensure UserRepository implements repository.UserRepository
var _ repository.UserRepository = (*UserRepository)(nil)
