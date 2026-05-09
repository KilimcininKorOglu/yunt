# Feature 2: Domain Models & Repository Layer

**Feature ID:** F002  
**Priority:** P1 - CRITICAL  
**Target Version:** v0.1.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature implements the core domain models and repository abstraction layer that forms the foundation of Yunt's data architecture. The domain models represent the business entities (User, Mailbox, Message, Attachment, Webhook) with all their properties and relationships. The repository layer provides a database-agnostic interface for data operations, enabling support for multiple database backends (SQLite, PostgreSQL, MySQL, MongoDB) without changing business logic.

The repository pattern ensures clean separation between business logic and data persistence, making the codebase maintainable and testable. This architecture allows developers to switch database backends based on their needs without modifying application code.

## Goals

- Define comprehensive domain models for all business entities
- Create a complete repository interface abstracting data operations
- Implement SQLite repository as the reference implementation
- Establish database migration framework
- Provide repository factory for dynamic backend selection
- Ensure data integrity and relationship management

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing
- [ ] Domain models accurately represent business requirements
- [ ] Repository interface covers all data operations
- [ ] SQLite implementation works correctly
- [ ] Migrations create correct database schema
- [ ] Unit tests achieve > 70% coverage
- [ ] Repository operations are transaction-safe

## Tasks

### T007: Define Domain Models and Entities

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Create Go structs for all domain entities including User, Mailbox, Message, Attachment, Webhook, and Settings. Include all properties, enums, and helper methods. Implement JSON marshaling/unmarshaling and validation logic.

#### Technical Details

- Define User entity with authentication fields
- Define Role enum (admin, user)
- Define Mailbox entity with type distinction (system, custom)
- Define Message entity with RFC 5322 compliant fields
- Define Address struct for email addresses
- Define Attachment entity with content handling
- Define Webhook entity with event types
- Add JSON struct tags for API serialization
- Implement String() methods for logging
- Add validation methods for each entity
- Define domain-specific errors

#### Files to Touch

- `internal/domain/user.go` (new)
- `internal/domain/mailbox.go` (new)
- `internal/domain/message.go` (new)
- `internal/domain/attachment.go` (new)
- `internal/domain/webhook.go` (new)
- `internal/domain/settings.go` (new)
- `internal/domain/errors.go` (new)
- `internal/domain/types.go` (new)

#### Dependencies

- T001 (project structure)

#### Success Criteria

- [ ] All domain structs defined with complete fields
- [ ] JSON marshaling works correctly
- [ ] Validation methods catch invalid data
- [ ] Domain errors are well-defined
- [ ] Struct tags are consistent
- [ ] Documentation comments explain each field
- [ ] Unit tests for validation logic pass

---

### T008: Design Repository Interface Layer

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Define comprehensive repository interfaces for all data operations. Create interfaces for UserRepository, MailboxRepository, MessageRepository, AttachmentRepository, WebhookRepository, and SettingsRepository. Include pagination, filtering, and search capabilities.

#### Technical Details

- Define main Repository interface with lifecycle methods
- Define UserRepository with CRUD and query operations
- Define MailboxRepository with user-specific operations
- Define MessageRepository with complex filtering and search
- Define AttachmentRepository with content management
- Define WebhookRepository for webhook configuration
- Define SettingsRepository for key-value storage
- Create ListOptions struct for pagination
- Create MessageListOptions for message filtering
- Create SearchQuery struct for advanced search
- Document all interface methods thoroughly

#### Files to Touch

- `internal/repository/repository.go` (new)
- `internal/repository/user.go` (new)
- `internal/repository/mailbox.go` (new)
- `internal/repository/message.go` (new)
- `internal/repository/attachment.go` (new)
- `internal/repository/webhook.go` (new)
- `internal/repository/settings.go` (new)
- `internal/repository/options.go` (new)

#### Dependencies

- T007 (domain models must exist)

#### Success Criteria

- [ ] All repository interfaces defined
- [ ] Interface methods are intuitive and consistent
- [ ] Pagination support is comprehensive
- [ ] Filtering options cover common use cases
- [ ] Search capabilities are flexible
- [ ] Context support for cancellation
- [ ] Error returns are consistent
- [ ] Interface documentation is complete

---

### T009: Implement SQLite Repository Foundation

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Implement the complete SQLite repository including connection management, schema creation, and basic CRUD operations for all entities. This serves as the reference implementation for other database backends.

#### Technical Details

- Use `mattn/go-sqlite3` driver with CGO
- Implement connection pooling and WAL mode
- Create SQLite-specific type conversions
- Handle NULL values appropriately
- Implement prepared statement caching
- Use transactions for multi-step operations
- Implement proper error wrapping
- Add connection health checks

#### Files to Touch

- `internal/repository/sqlite/sqlite.go` (new)
- `internal/repository/sqlite/connection.go` (new)
- `internal/repository/sqlite/user.go` (new)
- `internal/repository/sqlite/mailbox.go` (new)
- `internal/repository/sqlite/message.go` (new)
- `internal/repository/sqlite/attachment.go` (new)
- `internal/repository/sqlite/webhook.go` (new)
- `internal/repository/sqlite/settings.go` (new)

#### Dependencies

- T008 (repository interfaces must exist)
- T002 (configuration for database path)

#### Success Criteria

- [ ] Database connection establishes successfully
- [ ] All CRUD operations work correctly
- [ ] Transactions commit and rollback properly
- [ ] Foreign key constraints are enforced
- [ ] Concurrent access is safe
- [ ] Error messages are descriptive
- [ ] Connection pool manages resources efficiently
- [ ] Unit tests for all operations pass

---

### T010: Create Database Migration System

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 2 days

#### Description

Implement a database migration system for SQLite that creates tables, indexes, and initial data. Support migration versioning and rollback capabilities. Ensure migrations are idempotent.

#### Technical Details

- Create SQL schema matching PRD specifications
- Define tables: users, mailboxes, messages, attachments, webhooks, settings
- Create appropriate indexes for performance
- Set up foreign key relationships
- Implement migration versioning system
- Create schema_migrations table for tracking
- Support up/down migrations
- Create initial admin user from configuration
- Create default mailboxes (INBOX, Sent, Drafts, Trash, Spam)

#### Files to Touch

- `internal/repository/sqlite/migrations.go` (new)
- `internal/repository/sqlite/schema.sql` (new)
- `internal/repository/sqlite/migrations/001_initial_schema.sql` (new)
- `internal/repository/sqlite/migrations/002_indexes.sql` (new)
- `internal/repository/sqlite/seed.go` (new)

#### Dependencies

- T009 (SQLite repository foundation)

#### Success Criteria

- [ ] Schema creation SQL is valid
- [ ] All tables created with correct columns
- [ ] Indexes improve query performance
- [ ] Foreign keys enforce referential integrity
- [ ] Migration versioning works correctly
- [ ] Migrations are idempotent (can run multiple times)
- [ ] Admin user is created on first run
- [ ] Default mailboxes are created for new users

---

### T011: Implement Repository Factory Pattern

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create a repository factory that instantiates the correct repository implementation based on configuration. This enables dynamic database backend selection at runtime.

#### Technical Details

- Implement factory function based on database driver config
- Support driver types: sqlite, postgres, mysql, mongodb
- Return repository interface, not concrete type
- Validate configuration before creating repository
- Handle driver-specific initialization
- Implement graceful error handling for unsupported drivers
- Add factory unit tests with mock configuration

#### Files to Touch

- `internal/repository/factory.go` (new)
- `internal/repository/factory_test.go` (new)

#### Dependencies

- T008 (repository interfaces)
- T009 (SQLite implementation as first backend)

#### Success Criteria

- [ ] Factory creates correct repository based on config
- [ ] Invalid driver returns descriptive error
- [ ] Factory validates configuration before creation
- [ ] Returns interface, not concrete type
- [ ] Unit tests cover all driver types
- [ ] Error handling is comprehensive

---

### T012: Add Message and Mailbox Statistics

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Implement statistics calculation for mailboxes and messages. Include unread counts, starred counts, total message counts, and storage usage. Optimize queries for performance.

#### Technical Details

- Implement GetStats for MailboxRepository
- Implement GetStats for MessageRepository
- Use efficient COUNT queries with indexes
- Calculate storage usage from message sizes
- Support aggregation across mailboxes
- Cache statistics where appropriate
- Update counts transactionally with message operations
- Add denormalized counters to mailbox table

#### Files to Touch

- `internal/domain/stats.go` (new)
- `internal/repository/sqlite/mailbox.go` (update)
- `internal/repository/sqlite/message.go` (update)
- `internal/repository/sqlite/stats.go` (new)

#### Dependencies

- T009 (SQLite repository)
- T010 (migrations for counter columns)

#### Success Criteria

- [ ] Statistics queries are fast (< 10ms)
- [ ] Counts are accurate
- [ ] Statistics update with message changes
- [ ] Storage calculations are correct
- [ ] Denormalized counters stay in sync
- [ ] Unit tests verify accuracy

---

## Performance Targets

- Repository operation latency: < 10ms (p95)
- Bulk insert performance: > 1000 messages/second
- Search query performance: < 50ms for 10k messages
- Database connection pool: 5-25 connections
- Memory usage: < 100MB for 10k messages

## Risk Assessment

| Risk                              | Probability | Impact | Mitigation                                    |
|-----------------------------------|-------------|--------|-----------------------------------------------|
| SQLite performance limitations    | Medium      | Medium | Optimize indexes, implement caching           |
| Data consistency issues           | Low         | High   | Use transactions, enforce foreign keys        |
| Migration failures                | Low         | High   | Test thoroughly, implement rollback           |
| Repository interface changes      | Medium      | Medium | Design carefully upfront, version interfaces  |
| Concurrent write conflicts        | Medium      | Medium | Use proper locking, WAL mode for SQLite       |

## Notes

- SQLite is perfect for development but consider PostgreSQL for production
- Repository pattern enables easy testing with mock implementations
- Statistics should be eventually consistent, not real-time
- Consider implementing soft deletes for messages
- Binary data (attachments) may need special handling for large sizes
- Full-text search capabilities should be database-agnostic where possible
- Migration system should support both schema and data migrations
