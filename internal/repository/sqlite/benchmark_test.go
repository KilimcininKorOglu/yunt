// Package sqlite provides comprehensive benchmarks for SQLite database operations.
// These benchmarks test CRUD operations, search, bulk operations, and complex queries.
//
// Run all benchmarks:
//
//	go test -bench=. -benchmem ./internal/repository/sqlite/...
//
// Run specific benchmark:
//
//	go test -bench=BenchmarkUserCreate -benchmem ./internal/repository/sqlite/...
//
// Run with custom iteration time:
//
//	go test -bench=. -benchmem -benchtime=3s ./internal/repository/sqlite/...
//
// Generate results for comparison:
//
//	go test -bench=. -benchmem -count=5 ./internal/repository/sqlite/... | tee sqlite_results.txt
package sqlite

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"yunt/internal/domain"
	"yunt/internal/repository"
)

// benchRepo creates an in-memory SQLite repository for benchmarks.
func benchRepo(b *testing.B) *Repository {
	b.Helper()

	cfg := &ConnectionConfig{
		DSN:               ":memory:",
		MaxOpenConns:      1,
		MaxIdleConns:      1,
		EnableForeignKeys: true,
		JournalMode:       "MEMORY",
		SynchronousMode:   "OFF",
	}

	pool, err := NewConnectionPool(cfg)
	if err != nil {
		b.Fatalf("failed to create connection pool: %v", err)
	}

	repo, err := New(pool)
	if err != nil {
		pool.Close()
		b.Fatalf("failed to create repository: %v", err)
	}

	b.Cleanup(func() {
		repo.Close()
	})

	return repo
}

// benchUser creates a user for benchmark tests.
func benchUser(ctx context.Context, b *testing.B, repo *Repository) *domain.User {
	b.Helper()

	user := &domain.User{
		ID:           domain.ID("bench-setup-user"),
		Username:     "benchsetupuser",
		Email:        "benchsetup@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create setup user: %v", err)
	}
	return user
}

// benchMailbox creates a mailbox for benchmark tests.
func benchMailbox(ctx context.Context, b *testing.B, repo *Repository, user *domain.User) *domain.Mailbox {
	b.Helper()

	mailbox := &domain.Mailbox{
		ID:        domain.ID("bench-setup-mailbox"),
		UserID:    user.ID,
		Name:      "Benchmark Inbox",
		Address:   "benchsetup@test.local",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
		b.Fatalf("failed to create setup mailbox: %v", err)
	}
	return mailbox
}

// benchMessage creates a message for benchmark tests.
func benchMessage(ctx context.Context, b *testing.B, repo *Repository, mailbox *domain.Mailbox) *domain.Message {
	b.Helper()

	msg := &domain.Message{
		ID:          domain.ID("bench-setup-message"),
		MailboxID:   mailbox.ID,
		MessageID:   "<benchsetup@example.com>",
		From:        domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
		To:          []domain.EmailAddress{{Address: mailbox.Address}},
		Subject:     "Benchmark Setup Message",
		TextBody:    "This is a benchmark test message body.",
		ContentType: domain.ContentTypePlain,
		Size:        256,
		Status:      domain.MessageUnread,
		ReceivedAt:  domain.Now(),
		CreatedAt:   domain.Now(),
		UpdatedAt:   domain.Now(),
	}
	if err := repo.Messages().Create(ctx, msg); err != nil {
		b.Fatalf("failed to create setup message: %v", err)
	}
	return msg
}

// =============================================================================
// User Repository Benchmarks
// =============================================================================

// BenchmarkUserCreate benchmarks user creation operations.
func BenchmarkUserCreate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-user-%d-%d", b.N, i)),
			Username:     fmt.Sprintf("benchuser%d_%d", b.N, i),
			Email:        fmt.Sprintf("benchuser%d_%d@example.com", b.N, i),
			PasswordHash: "hashedpassword123",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}
}

// BenchmarkUserGetByID benchmarks user retrieval by ID.
func BenchmarkUserGetByID(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	user := &domain.User{
		ID:           domain.ID("bench-getbyid-user"),
		Username:     "benchgetuser",
		Email:        "benchget@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().GetByID(ctx, user.ID); err != nil {
			b.Fatalf("failed to get user: %v", err)
		}
	}
}

// BenchmarkUserGetByUsername benchmarks user retrieval by username.
func BenchmarkUserGetByUsername(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	user := &domain.User{
		ID:           domain.ID("bench-getbyusername-user"),
		Username:     "benchgetbyusername",
		Email:        "benchgetbyusername@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().GetByUsername(ctx, user.Username); err != nil {
			b.Fatalf("failed to get user by username: %v", err)
		}
	}
}

// BenchmarkUserGetByEmail benchmarks user retrieval by email.
func BenchmarkUserGetByEmail(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	user := &domain.User{
		ID:           domain.ID("bench-getbyemail-user"),
		Username:     "benchgetbyemail",
		Email:        "benchgetbyemail@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().GetByEmail(ctx, user.Email); err != nil {
			b.Fatalf("failed to get user by email: %v", err)
		}
	}
}

// BenchmarkUserUpdate benchmarks user update operations.
func BenchmarkUserUpdate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	user := &domain.User{
		ID:           domain.ID("bench-update-user"),
		Username:     "benchupdateuser",
		Email:        "benchupdate@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user.DisplayName = fmt.Sprintf("Updated Name %d", i)
		user.UpdatedAt = domain.Now()
		if err := repo.Users().Update(ctx, user); err != nil {
			b.Fatalf("failed to update user: %v", err)
		}
	}
}

// BenchmarkUserList benchmarks user listing with pagination.
func BenchmarkUserList(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Setup: create 100 users for listing
	for i := 0; i < 100; i++ {
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-list-user-%d", i)),
			Username:     fmt.Sprintf("benchlistuser%d", i),
			Email:        fmt.Sprintf("benchlist%d@example.com", i),
			PasswordHash: "hashedpassword123",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().List(ctx, nil, opts); err != nil {
			b.Fatalf("failed to list users: %v", err)
		}
	}
}

// BenchmarkUserSearch benchmarks user search operations.
func BenchmarkUserSearch(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Setup: create 100 users with varied data
	for i := 0; i < 100; i++ {
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-search-user-%d", i)),
			Username:     fmt.Sprintf("searchuser%d", i),
			Email:        fmt.Sprintf("search%d@example.com", i),
			DisplayName:  fmt.Sprintf("Search User Number %d", i),
			PasswordHash: "hashedpassword123",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().Search(ctx, "search", opts); err != nil {
			b.Fatalf("failed to search users: %v", err)
		}
	}
}

// BenchmarkUserCount benchmarks user count operations.
func BenchmarkUserCount(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Setup: create 100 users
	for i := 0; i < 100; i++ {
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-count-user-%d", i)),
			Username:     fmt.Sprintf("countuser%d", i),
			Email:        fmt.Sprintf("count%d@example.com", i),
			PasswordHash: "hashedpassword123",
			Role:         domain.RoleUser,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().Count(ctx, nil); err != nil {
			b.Fatalf("failed to count users: %v", err)
		}
	}
}

// BenchmarkUserExists benchmarks user existence check.
func BenchmarkUserExists(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	user := &domain.User{
		ID:           domain.ID("bench-exists-user"),
		Username:     "benchexistsuser",
		Email:        "benchexists@example.com",
		PasswordHash: "hashedpassword123",
		Role:         domain.RoleUser,
		Status:       domain.StatusActive,
		CreatedAt:    domain.Now(),
		UpdatedAt:    domain.Now(),
	}
	if err := repo.Users().Create(ctx, user); err != nil {
		b.Fatalf("failed to create user: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().Exists(ctx, user.ID); err != nil {
			b.Fatalf("failed to check user existence: %v", err)
		}
	}
}

// BenchmarkUserCountByRole benchmarks user count by role aggregation.
func BenchmarkUserCountByRole(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Create 50 users with different roles
	for i := 0; i < 50; i++ {
		role := domain.RoleUser
		if i%5 == 0 {
			role = domain.RoleAdmin
		}
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-countbyrole-user-%d", i)),
			Username:     fmt.Sprintf("roleuser%d", i),
			Email:        fmt.Sprintf("role%d@example.com", i),
			PasswordHash: "hash",
			Role:         role,
			Status:       domain.StatusActive,
			CreatedAt:    domain.Now(),
			UpdatedAt:    domain.Now(),
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().CountByRole(ctx); err != nil {
			b.Fatalf("failed to count by role: %v", err)
		}
	}
}

// =============================================================================
// Mailbox Repository Benchmarks
// =============================================================================

// BenchmarkMailboxCreate benchmarks mailbox creation.
func BenchmarkMailboxCreate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mailbox := &domain.Mailbox{
			ID:        domain.ID(fmt.Sprintf("bench-mailbox-%d-%d", b.N, i)),
			UserID:    user.ID,
			Name:      fmt.Sprintf("Inbox %d", i),
			Address:   fmt.Sprintf("inbox%d_%d@example.com", b.N, i),
			CreatedAt: domain.Now(),
			UpdatedAt: domain.Now(),
		}
		if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
			b.Fatalf("failed to create mailbox: %v", err)
		}
	}
}

// BenchmarkMailboxGetByID benchmarks mailbox retrieval by ID.
func BenchmarkMailboxGetByID(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("bench-getbyid-mailbox"),
		UserID:    user.ID,
		Name:      "Benchmark Inbox",
		Address:   "benchgetbyid@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
		b.Fatalf("failed to create mailbox: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Mailboxes().GetByID(ctx, mailbox.ID); err != nil {
			b.Fatalf("failed to get mailbox: %v", err)
		}
	}
}

// BenchmarkMailboxGetByAddress benchmarks mailbox retrieval by address.
func BenchmarkMailboxGetByAddress(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("bench-getbyaddr-mailbox"),
		UserID:    user.ID,
		Name:      "Benchmark Inbox",
		Address:   "benchgetbyaddr@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
		b.Fatalf("failed to create mailbox: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Mailboxes().GetByAddress(ctx, mailbox.Address); err != nil {
			b.Fatalf("failed to get mailbox by address: %v", err)
		}
	}
}

// BenchmarkMailboxListByUser benchmarks listing mailboxes by user.
func BenchmarkMailboxListByUser(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	// Create 20 mailboxes for the user
	for i := 0; i < 20; i++ {
		mailbox := &domain.Mailbox{
			ID:        domain.ID(fmt.Sprintf("bench-listbyuser-mailbox-%d", i)),
			UserID:    user.ID,
			Name:      fmt.Sprintf("Mailbox %d", i),
			Address:   fmt.Sprintf("listbyuser%d@example.com", i),
			CreatedAt: domain.Now(),
			UpdatedAt: domain.Now(),
		}
		if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
			b.Fatalf("failed to create mailbox: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Mailboxes().ListByUser(ctx, user.ID, nil); err != nil {
			b.Fatalf("failed to list mailboxes: %v", err)
		}
	}
}

// BenchmarkMailboxUpdateStats benchmarks mailbox stats updates.
func BenchmarkMailboxUpdateStats(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	mailbox := &domain.Mailbox{
		ID:        domain.ID("bench-updatestats-mailbox"),
		UserID:    user.ID,
		Name:      "Stats Mailbox",
		Address:   "benchupdatestats@example.com",
		CreatedAt: domain.Now(),
		UpdatedAt: domain.Now(),
	}
	if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
		b.Fatalf("failed to create mailbox: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Mailboxes().IncrementMessageCount(ctx, mailbox.ID, 1024, true); err != nil {
			b.Fatalf("failed to update stats: %v", err)
		}
	}
}

// BenchmarkMailboxGetTotalStats benchmarks getting total mailbox statistics.
func BenchmarkMailboxGetTotalStats(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	// Create 10 mailboxes with messages
	for i := 0; i < 10; i++ {
		mailbox := &domain.Mailbox{
			ID:        domain.ID(fmt.Sprintf("bench-totalstats-mailbox-%d", i)),
			UserID:    user.ID,
			Name:      fmt.Sprintf("Mailbox %d", i),
			Address:   fmt.Sprintf("totalstats%d@example.com", i),
			CreatedAt: domain.Now(),
			UpdatedAt: domain.Now(),
		}
		if err := repo.Mailboxes().Create(ctx, mailbox); err != nil {
			b.Fatalf("failed to create mailbox: %v", err)
		}

		// Create some messages
		for j := 0; j < 10; j++ {
			msg := &domain.Message{
				ID:          domain.ID(fmt.Sprintf("bench-totalstats-msg-%d-%d", i, j)),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<totalstats%d_%d@example.com>", i, j),
				From:        domain.EmailAddress{Address: "sender@example.com"},
				To:          []domain.EmailAddress{{Address: mailbox.Address}},
				Subject:     "Stats Test",
				ContentType: domain.ContentTypePlain,
				Size:        512,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			if err := repo.Messages().Create(ctx, msg); err != nil {
				b.Fatalf("failed to create message: %v", err)
			}
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Mailboxes().GetTotalStats(ctx); err != nil {
			b.Fatalf("failed to get total stats: %v", err)
		}
	}
}

// =============================================================================
// Message Repository Benchmarks
// =============================================================================

// BenchmarkMessageCreate benchmarks message creation.
func BenchmarkMessageCreate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-msg-%d-%d", b.N, i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<bench%d_%d@example.com>", b.N, i),
			From:        domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("Benchmark Message %d", i),
			TextBody:    "This is a benchmark test message body.",
			ContentType: domain.ContentTypePlain,
			Size:        256,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
	}
}

// BenchmarkMessageGetByID benchmarks message retrieval by ID.
func BenchmarkMessageGetByID(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().GetByID(ctx, msg.ID); err != nil {
			b.Fatalf("failed to get message: %v", err)
		}
	}
}

// BenchmarkMessageGetByMessageID benchmarks message retrieval by Message-ID header.
func BenchmarkMessageGetByMessageID(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().GetByMessageID(ctx, msg.MessageID); err != nil {
			b.Fatalf("failed to get message by Message-ID: %v", err)
		}
	}
}

// BenchmarkMessageListByMailbox benchmarks listing messages in a mailbox.
func BenchmarkMessageListByMailbox(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	// Create 100 messages
	for i := 0; i < 100; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-listbymailbox-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<listbymailbox%d@example.com>", i),
			From:        domain.EmailAddress{Name: "Sender", Address: "sender@example.com"},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("Test Message %d", i),
			TextBody:    "Test body",
			ContentType: domain.ContentTypePlain,
			Size:        256,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
	}

	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().ListByMailbox(ctx, mailbox.ID, opts); err != nil {
			b.Fatalf("failed to list messages: %v", err)
		}
	}
}

// BenchmarkMessageSearch benchmarks message search operations.
func BenchmarkMessageSearch(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	// Create 100 messages with varied content
	subjects := []string{"Important", "Meeting", "Report", "Invoice", "Newsletter"}
	for i := 0; i < 100; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-search-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<search%d@example.com>", i),
			From:        domain.EmailAddress{Name: "Sender", Address: fmt.Sprintf("sender%d@example.com", i%10)},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("%s Message %d", subjects[i%len(subjects)], i),
			TextBody:    fmt.Sprintf("This is a test message with content about topic %d.", i),
			ContentType: domain.ContentTypePlain,
			Size:        512,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
	}

	searchOpts := &repository.SearchOptions{Query: "important"}
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().Search(ctx, searchOpts, nil, opts); err != nil {
			b.Fatalf("failed to search messages: %v", err)
		}
	}
}

// BenchmarkMessageMarkAsRead benchmarks marking messages as read.
func BenchmarkMessageMarkAsRead(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	// Create messages for the benchmark
	msgs := make([]*domain.Message, b.N)
	for i := 0; i < b.N; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-markread-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<markread%d@example.com>", i),
			From:        domain.EmailAddress{Address: "sender@example.com"},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("Mark Read Test %d", i),
			ContentType: domain.ContentTypePlain,
			Size:        100,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
		msgs[i] = msg
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().MarkAsRead(ctx, msgs[i].ID); err != nil {
			b.Fatalf("failed to mark as read: %v", err)
		}
	}
}

// BenchmarkMessageStar benchmarks starring messages.
func BenchmarkMessageStar(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			if err := repo.Messages().Star(ctx, msg.ID); err != nil {
				b.Fatalf("failed to star: %v", err)
			}
		} else {
			if err := repo.Messages().Unstar(ctx, msg.ID); err != nil {
				b.Fatalf("failed to unstar: %v", err)
			}
		}
	}
}

// BenchmarkMessageCountByMailbox benchmarks counting messages in a mailbox.
func BenchmarkMessageCountByMailbox(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	// Create 100 messages
	for i := 0; i < 100; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-count-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<count%d@example.com>", i),
			From:        domain.EmailAddress{Address: "sender@example.com"},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("Count Test %d", i),
			ContentType: domain.ContentTypePlain,
			Size:        100,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().CountByMailbox(ctx, mailbox.ID); err != nil {
			b.Fatalf("failed to count: %v", err)
		}
	}
}

// BenchmarkMessageBulkMarkAsRead benchmarks bulk mark as read operations.
func BenchmarkMessageBulkMarkAsRead(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create 50 messages for bulk operation
		ids := make([]domain.ID, 50)
		for j := 0; j < 50; j++ {
			msg := &domain.Message{
				ID:          domain.ID(fmt.Sprintf("bench-bulkread-%d-%d", i, j)),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<bulkread%d_%d@example.com>", i, j),
				From:        domain.EmailAddress{Address: "sender@example.com"},
				To:          []domain.EmailAddress{{Address: mailbox.Address}},
				Subject:     "Bulk Read Test",
				ContentType: domain.ContentTypePlain,
				Size:        100,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			if err := repo.Messages().Create(ctx, msg); err != nil {
				b.Fatalf("failed to create message: %v", err)
			}
			ids[j] = msg.ID
		}
		b.StartTimer()

		if _, err := repo.Messages().BulkMarkAsRead(ctx, ids); err != nil {
			b.Fatalf("failed to bulk mark as read: %v", err)
		}
	}
}

// BenchmarkComplexMessageFilter benchmarks message listing with complex filters.
func BenchmarkComplexMessageFilter(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)

	// Create 200 messages with varied properties
	now := domain.Now()
	for i := 0; i < 200; i++ {
		receivedAt := domain.Timestamp{Time: now.Time.Add(-time.Duration(i) * time.Hour)}
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-complexfilter-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<complexfilter%d@example.com>", i),
			From:        domain.EmailAddress{Name: "Sender", Address: fmt.Sprintf("sender%d@example.com", i%10)},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     fmt.Sprintf("Complex Filter Message %d", i),
			TextBody:    "Test content for filtering",
			ContentType: domain.ContentTypePlain,
			Size:        int64(100 + i*10),
			Status:      domain.MessageUnread,
			IsStarred:   i%3 == 0,
			IsSpam:      i%7 == 0,
			ReceivedAt:  receivedAt,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to create message: %v", err)
		}
	}

	starred := true
	minSize := int64(500)
	filter := &repository.MessageFilter{
		MailboxID: &mailbox.ID,
		IsStarred: &starred,
		MinSize:   &minSize,
	}
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
		Sort: &repository.SortOptions{
			Field: "receivedAt",
			Order: domain.SortDesc,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Messages().List(ctx, filter, opts); err != nil {
			b.Fatalf("failed to list with filter: %v", err)
		}
	}
}

// =============================================================================
// Attachment Repository Benchmarks
// =============================================================================

// BenchmarkAttachmentCreate benchmarks attachment creation.
func BenchmarkAttachmentCreate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		att := &domain.Attachment{
			ID:          domain.ID(fmt.Sprintf("bench-att-%d-%d", b.N, i)),
			MessageID:   msg.ID,
			Filename:    fmt.Sprintf("document%d.pdf", i),
			ContentType: "application/pdf",
			Size:        1024,
			Disposition: domain.DispositionAttachment,
			CreatedAt:   domain.Now(),
		}
		if err := repo.Attachments().Create(ctx, att); err != nil {
			b.Fatalf("failed to create attachment: %v", err)
		}
	}
}

// BenchmarkAttachmentGetByID benchmarks attachment retrieval.
func BenchmarkAttachmentGetByID(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	att := &domain.Attachment{
		ID:          domain.ID("bench-getbyid-att"),
		MessageID:   msg.ID,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
		Size:        1024,
		Disposition: domain.DispositionAttachment,
		CreatedAt:   domain.Now(),
	}
	if err := repo.Attachments().Create(ctx, att); err != nil {
		b.Fatalf("failed to create attachment: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Attachments().GetByID(ctx, att.ID); err != nil {
			b.Fatalf("failed to get attachment: %v", err)
		}
	}
}

// BenchmarkAttachmentListByMessage benchmarks listing attachments by message.
func BenchmarkAttachmentListByMessage(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	msg := benchMessage(ctx, b, repo, mailbox)

	// Create 5 attachments
	for i := 0; i < 5; i++ {
		att := &domain.Attachment{
			ID:          domain.ID(fmt.Sprintf("bench-listbymsg-att-%d", i)),
			MessageID:   msg.ID,
			Filename:    fmt.Sprintf("file%d.pdf", i),
			ContentType: "application/pdf",
			Size:        1024,
			Disposition: domain.DispositionAttachment,
			CreatedAt:   domain.Now(),
		}
		if err := repo.Attachments().Create(ctx, att); err != nil {
			b.Fatalf("failed to create attachment: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Attachments().ListByMessage(ctx, msg.ID); err != nil {
			b.Fatalf("failed to list attachments: %v", err)
		}
	}
}

// =============================================================================
// Webhook Repository Benchmarks
// =============================================================================

// BenchmarkWebhookCreate benchmarks webhook creation.
func BenchmarkWebhookCreate(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		webhook := &domain.Webhook{
			ID:             domain.ID(fmt.Sprintf("bench-webhook-%d-%d", b.N, i)),
			UserID:         user.ID,
			Name:           fmt.Sprintf("Benchmark Webhook %d", i),
			URL:            fmt.Sprintf("https://example.com/webhook%d", i),
			Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
			Status:         domain.WebhookStatusActive,
			MaxRetries:     3,
			TimeoutSeconds: 30,
			CreatedAt:      domain.Now(),
			UpdatedAt:      domain.Now(),
		}
		if err := repo.Webhooks().Create(ctx, webhook); err != nil {
			b.Fatalf("failed to create webhook: %v", err)
		}
	}
}

// BenchmarkWebhookListByUser benchmarks listing webhooks by user.
func BenchmarkWebhookListByUser(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	// Create 10 webhooks
	for i := 0; i < 10; i++ {
		webhook := &domain.Webhook{
			ID:             domain.ID(fmt.Sprintf("bench-listbyuser-webhook-%d", i)),
			UserID:         user.ID,
			Name:           fmt.Sprintf("Webhook %d", i),
			URL:            fmt.Sprintf("https://example.com/hook%d", i),
			Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
			Status:         domain.WebhookStatusActive,
			MaxRetries:     3,
			TimeoutSeconds: 30,
			CreatedAt:      domain.Now(),
			UpdatedAt:      domain.Now(),
		}
		if err := repo.Webhooks().Create(ctx, webhook); err != nil {
			b.Fatalf("failed to create webhook: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Webhooks().ListByUser(ctx, user.ID, nil); err != nil {
			b.Fatalf("failed to list webhooks: %v", err)
		}
	}
}

// BenchmarkWebhookListActiveByEvent benchmarks listing active webhooks by event.
func BenchmarkWebhookListActiveByEvent(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	// Create 20 webhooks with different events
	events := []domain.WebhookEvent{
		domain.WebhookEventMessageReceived,
		domain.WebhookEventMessageDeleted,
	}
	for i := 0; i < 20; i++ {
		webhook := &domain.Webhook{
			ID:             domain.ID(fmt.Sprintf("bench-listbyevent-webhook-%d", i)),
			UserID:         user.ID,
			Name:           fmt.Sprintf("Webhook %d", i),
			URL:            fmt.Sprintf("https://example.com/hook%d", i),
			Events:         []domain.WebhookEvent{events[i%len(events)]},
			Status:         domain.WebhookStatusActive,
			MaxRetries:     3,
			TimeoutSeconds: 30,
			CreatedAt:      domain.Now(),
			UpdatedAt:      domain.Now(),
		}
		if err := repo.Webhooks().Create(ctx, webhook); err != nil {
			b.Fatalf("failed to create webhook: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Webhooks().ListActiveByEvent(ctx, domain.WebhookEventMessageReceived); err != nil {
			b.Fatalf("failed to list webhooks by event: %v", err)
		}
	}
}

// BenchmarkWebhookRecordSuccess benchmarks recording webhook success.
func BenchmarkWebhookRecordSuccess(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	webhook := &domain.Webhook{
		ID:             domain.ID("bench-recordsuccess-webhook"),
		UserID:         user.ID,
		Name:           "Success Webhook",
		URL:            "https://example.com/success",
		Events:         []domain.WebhookEvent{domain.WebhookEventMessageReceived},
		Status:         domain.WebhookStatusActive,
		MaxRetries:     3,
		TimeoutSeconds: 30,
		CreatedAt:      domain.Now(),
		UpdatedAt:      domain.Now(),
	}
	if err := repo.Webhooks().Create(ctx, webhook); err != nil {
		b.Fatalf("failed to create webhook: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := repo.Webhooks().RecordSuccess(ctx, webhook.ID); err != nil {
			b.Fatalf("failed to record success: %v", err)
		}
	}
}

// =============================================================================
// Transaction Benchmarks
// =============================================================================

// BenchmarkTransaction benchmarks transaction execution.
func BenchmarkTransaction(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.Transaction(ctx, func(tx repository.Repository) error {
			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("bench-tx-user-%d", i)),
				Username:     fmt.Sprintf("txuser%d", i),
				Email:        fmt.Sprintf("tx%d@example.com", i),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			return tx.Users().Create(ctx, user)
		})
		if err != nil {
			b.Fatalf("transaction failed: %v", err)
		}
	}
}

// BenchmarkTransactionWithMultipleOps benchmarks transactions with multiple operations.
func BenchmarkTransactionWithMultipleOps(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := repo.Transaction(ctx, func(tx repository.Repository) error {
			// Create user
			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("bench-txmulti-user-%d", i)),
				Username:     fmt.Sprintf("txmultiuser%d", i),
				Email:        fmt.Sprintf("txmulti%d@example.com", i),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			if err := tx.Users().Create(ctx, user); err != nil {
				return err
			}

			// Create mailbox
			mailbox := &domain.Mailbox{
				ID:        domain.ID(fmt.Sprintf("bench-txmulti-mailbox-%d", i)),
				UserID:    user.ID,
				Name:      "Inbox",
				Address:   fmt.Sprintf("txmulti%d@test.local", i),
				CreatedAt: domain.Now(),
				UpdatedAt: domain.Now(),
			}
			if err := tx.Mailboxes().Create(ctx, mailbox); err != nil {
				return err
			}

			// Create message
			msg := &domain.Message{
				ID:          domain.ID(fmt.Sprintf("bench-txmulti-msg-%d", i)),
				MailboxID:   mailbox.ID,
				MessageID:   fmt.Sprintf("<txmulti%d@example.com>", i),
				From:        domain.EmailAddress{Address: "sender@example.com"},
				To:          []domain.EmailAddress{{Address: mailbox.Address}},
				Subject:     "Transaction Test",
				ContentType: domain.ContentTypePlain,
				Size:        100,
				Status:      domain.MessageUnread,
				ReceivedAt:  domain.Now(),
				CreatedAt:   domain.Now(),
				UpdatedAt:   domain.Now(),
			}
			return tx.Messages().Create(ctx, msg)
		})
		if err != nil {
			b.Fatalf("transaction failed: %v", err)
		}
	}
}

// =============================================================================
// Concurrent Access Benchmarks
// =============================================================================

// BenchmarkConcurrentReads benchmarks concurrent read operations.
func BenchmarkConcurrentReads(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)
	user := benchUser(ctx, b, repo)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := repo.Users().GetByID(ctx, user.ID); err != nil {
				b.Errorf("failed to get user: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentWrites benchmarks concurrent write operations.
func BenchmarkConcurrentWrites(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			id := counter
			mu.Unlock()

			user := &domain.User{
				ID:           domain.ID(fmt.Sprintf("bench-concurrent-user-%d", id)),
				Username:     fmt.Sprintf("concurrentuser%d", id),
				Email:        fmt.Sprintf("concurrent%d@example.com", id),
				PasswordHash: "hash",
				Role:         domain.RoleUser,
				Status:       domain.StatusActive,
				CreatedAt:    domain.Now(),
				UpdatedAt:    domain.Now(),
			}
			if err := repo.Users().Create(ctx, user); err != nil {
				b.Errorf("failed to create user: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentMixedOps benchmarks mixed read/write concurrent operations.
func BenchmarkConcurrentMixedOps(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Setup some initial data
	user := benchUser(ctx, b, repo)
	mailbox := benchMailbox(ctx, b, repo, user)
	for i := 0; i < 50; i++ {
		msg := &domain.Message{
			ID:          domain.ID(fmt.Sprintf("bench-mixedops-setup-msg-%d", i)),
			MailboxID:   mailbox.ID,
			MessageID:   fmt.Sprintf("<mixedops%d@example.com>", i),
			From:        domain.EmailAddress{Address: "sender@example.com"},
			To:          []domain.EmailAddress{{Address: mailbox.Address}},
			Subject:     "Mixed Ops Setup",
			ContentType: domain.ContentTypePlain,
			Size:        100,
			Status:      domain.MessageUnread,
			ReceivedAt:  domain.Now(),
			CreatedAt:   domain.Now(),
			UpdatedAt:   domain.Now(),
		}
		if err := repo.Messages().Create(ctx, msg); err != nil {
			b.Fatalf("failed to setup message: %v", err)
		}
	}

	var counter int64
	var mu sync.Mutex

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mu.Lock()
			counter++
			op := counter % 5
			id := counter
			mu.Unlock()

			switch op {
			case 0: // Read user
				if _, err := repo.Users().GetByID(ctx, user.ID); err != nil {
					b.Errorf("failed to get user: %v", err)
				}
			case 1: // List mailbox messages
				opts := &repository.ListOptions{Pagination: &repository.PaginationOptions{Page: 1, PerPage: 10}}
				if _, err := repo.Messages().ListByMailbox(ctx, mailbox.ID, opts); err != nil {
					b.Errorf("failed to list messages: %v", err)
				}
			case 2: // Create message
				msg := &domain.Message{
					ID:          domain.ID(fmt.Sprintf("bench-mixedops-msg-%d", id)),
					MailboxID:   mailbox.ID,
					MessageID:   fmt.Sprintf("<mixedopsrun%d@example.com>", id),
					From:        domain.EmailAddress{Address: "sender@example.com"},
					To:          []domain.EmailAddress{{Address: mailbox.Address}},
					Subject:     "Mixed Ops Run",
					ContentType: domain.ContentTypePlain,
					Size:        100,
					Status:      domain.MessageUnread,
					ReceivedAt:  domain.Now(),
					CreatedAt:   domain.Now(),
					UpdatedAt:   domain.Now(),
				}
				if err := repo.Messages().Create(ctx, msg); err != nil {
					b.Errorf("failed to create message: %v", err)
				}
			case 3: // Count messages
				if _, err := repo.Messages().CountByMailbox(ctx, mailbox.ID); err != nil {
					b.Errorf("failed to count: %v", err)
				}
			case 4: // Search messages
				searchOpts := &repository.SearchOptions{Query: "mixed"}
				opts := &repository.ListOptions{Pagination: &repository.PaginationOptions{Page: 1, PerPage: 10}}
				if _, err := repo.Messages().Search(ctx, searchOpts, nil, opts); err != nil {
					b.Errorf("failed to search: %v", err)
				}
			}
		}
	})
}

// =============================================================================
// Settings Repository Benchmarks
// =============================================================================

// BenchmarkSettingsGet benchmarks settings retrieval.
func BenchmarkSettingsGet(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Settings().Get(ctx); err != nil {
			b.Fatalf("failed to get settings: %v", err)
		}
	}
}

// BenchmarkSettingsSave benchmarks settings save operations.
func BenchmarkSettingsSave(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	settings, err := repo.Settings().Get(ctx)
	if err != nil {
		b.Fatalf("failed to get initial settings: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		settings.UpdatedAt = domain.Now()
		if err := repo.Settings().Save(ctx, settings); err != nil {
			b.Fatalf("failed to save settings: %v", err)
		}
	}
}

// =============================================================================
// Complex Query Benchmarks
// =============================================================================

// BenchmarkComplexUserFilter benchmarks user listing with complex filters.
func BenchmarkComplexUserFilter(b *testing.B) {
	ctx := context.Background()
	repo := benchRepo(b)

	// Create 100 users with varied properties
	now := domain.Now()
	for i := 0; i < 100; i++ {
		role := domain.RoleUser
		if i%10 == 0 {
			role = domain.RoleAdmin
		}
		status := domain.StatusActive
		if i%5 == 0 {
			status = domain.StatusInactive
		}
		user := &domain.User{
			ID:           domain.ID(fmt.Sprintf("bench-complexuserfilter-user-%d", i)),
			Username:     fmt.Sprintf("complexuser%d", i),
			Email:        fmt.Sprintf("complex%d@example.com", i),
			DisplayName:  fmt.Sprintf("Complex User %d", i),
			PasswordHash: "hash",
			Role:         role,
			Status:       status,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := repo.Users().Create(ctx, user); err != nil {
			b.Fatalf("failed to create user: %v", err)
		}
	}

	status := domain.StatusActive
	role := domain.RoleUser
	filter := &repository.UserFilter{
		Status: &status,
		Role:   &role,
		Search: "complex",
	}
	opts := &repository.ListOptions{
		Pagination: &repository.PaginationOptions{
			Page:    1,
			PerPage: 20,
		},
		Sort: &repository.SortOptions{
			Field: "createdAt",
			Order: domain.SortDesc,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := repo.Users().List(ctx, filter, opts); err != nil {
			b.Fatalf("failed to list with filter: %v", err)
		}
	}
}
