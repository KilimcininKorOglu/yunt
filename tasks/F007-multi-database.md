# Feature 7: Multi-Database Support

**Feature ID:** F007  
**Priority:** P2 - HIGH  
**Target Version:** v0.6.0  
**Estimated Duration:** 2-3 weeks  
**Status:** NOT_STARTED

## Overview

This feature extends Yunt's database support beyond SQLite to include PostgreSQL, MySQL, and MongoDB. It implements repository adapters for each database backend while maintaining a consistent interface for the application layer. The multi-database architecture enables users to choose the most appropriate database for their deployment scenario—SQLite for simplicity and development, PostgreSQL for production, MySQL for existing infrastructure, or MongoDB for document-oriented storage.

Each database implementation must provide identical functionality through the repository interface, ensuring that switching databases requires only configuration changes without code modifications. This architecture demonstrates clean separation between business logic and data persistence.

## Goals

- Implement PostgreSQL repository with full feature parity
- Implement MySQL repository with full feature parity
- Implement MongoDB repository with full feature parity
- Create database migration system for each backend
- Ensure consistent behavior across all databases
- Support database-specific optimizations
- Provide clear migration paths between databases
- Document performance characteristics of each backend

## Success Criteria

- [ ] All tasks completed
- [ ] All tests passing for each database
- [ ] Feature parity across all databases
- [ ] Migrations work correctly for each backend
- [ ] Performance meets targets for each database
- [ ] Integration tests pass with all databases
- [ ] Documentation covers all database options
- [ ] Configuration examples provided for each

## Tasks

### T050: Implement PostgreSQL Repository

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Implement complete PostgreSQL repository including connection management, all CRUD operations, full-text search using tsvector, and PostgreSQL-specific optimizations. Ensure feature parity with SQLite implementation.

#### Technical Details

- Use `lib/pq` driver for PostgreSQL
- Implement connection pooling with configuration
- Create all repository interfaces (User, Mailbox, Message, etc.)
- Use PostgreSQL-specific types (JSONB for metadata, TEXT[] for arrays)
- Implement full-text search with tsvector and tsquery
- Create GIN indexes for full-text search
- Use RETURNING clause for insert operations
- Implement efficient pagination with LIMIT/OFFSET
- Handle NULL values correctly
- Use transactions for multi-step operations
- Implement proper error handling and wrapping

#### Files to Touch

- `internal/repository/postgres/postgres.go` (new)
- `internal/repository/postgres/connection.go` (new)
- `internal/repository/postgres/user.go` (new)
- `internal/repository/postgres/mailbox.go` (new)
- `internal/repository/postgres/message.go` (new)
- `internal/repository/postgres/attachment.go` (new)
- `internal/repository/postgres/webhook.go` (new)
- `internal/repository/postgres/settings.go` (new)
- `internal/repository/postgres/search.go` (new)

#### Dependencies

- T008 (repository interfaces)
- T009 (SQLite reference implementation)
- T006 (PostgreSQL driver dependency)

#### Success Criteria

- [ ] All repository interfaces implemented
- [ ] Connection pooling works correctly
- [ ] Full-text search performs well
- [ ] All CRUD operations work
- [ ] Transactions commit and rollback properly
- [ ] Foreign keys enforced
- [ ] Integration tests pass
- [ ] Performance meets targets

---

### T051: Create PostgreSQL Migration System

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1.5 days

#### Description

Create database migration system for PostgreSQL including schema creation, indexes, constraints, and initial data. Support migration versioning and ensure migrations are idempotent.

#### Technical Details

- Create SQL schema for PostgreSQL
- Define all tables with appropriate column types
- Use TEXT instead of VARCHAR for flexibility
- Use BYTEA for binary data (raw messages, attachments)
- Create indexes for performance (B-tree, GIN for full-text)
- Set up foreign key constraints with CASCADE
- Create full-text search indexes (GIN on tsvector)
- Implement migration versioning system
- Create schema_migrations table
- Support up/down migrations
- Create initial admin user
- Create default mailboxes for new users

#### Files to Touch

- `internal/repository/postgres/migrations.go` (new)
- `internal/repository/postgres/migrations/001_initial_schema.sql` (new)
- `internal/repository/postgres/migrations/002_indexes.sql` (new)
- `internal/repository/postgres/migrations/003_full_text_search.sql` (new)
- `internal/repository/postgres/seed.go` (new)

#### Dependencies

- T050 (PostgreSQL repository)

#### Success Criteria

- [ ] Schema creates all tables correctly
- [ ] Indexes improve query performance
- [ ] Foreign keys enforce integrity
- [ ] Full-text search indexes work
- [ ] Migrations are versioned
- [ ] Migrations are idempotent
- [ ] Admin user created on first run
- [ ] Default mailboxes created

---

### T052: Implement MySQL Repository

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 3 days

#### Description

Implement complete MySQL repository including connection management, all CRUD operations, full-text search using MySQL FULLTEXT indexes, and MySQL-specific optimizations. Ensure feature parity with SQLite and PostgreSQL.

#### Technical Details

- Use `go-sql-driver/mysql` for MySQL
- Implement connection pooling with configuration
- Create all repository interfaces
- Use MySQL-specific types (LONGTEXT, LONGBLOB)
- Implement full-text search with MATCH AGAINST
- Create FULLTEXT indexes for search
- Handle MySQL's unique NULL handling
- Use InnoDB engine for transactions
- Set character set to utf8mb4 for full Unicode
- Parse parseTime=true for time handling
- Implement efficient pagination
- Handle MySQL-specific error codes

#### Files to Touch

- `internal/repository/mysql/mysql.go` (new)
- `internal/repository/mysql/connection.go` (new)
- `internal/repository/mysql/user.go` (new)
- `internal/repository/mysql/mailbox.go` (new)
- `internal/repository/mysql/message.go` (new)
- `internal/repository/mysql/attachment.go` (new)
- `internal/repository/mysql/webhook.go` (new)
- `internal/repository/mysql/settings.go` (new)
- `internal/repository/mysql/search.go` (new)

#### Dependencies

- T008 (repository interfaces)
- T009 (SQLite reference implementation)
- T006 (MySQL driver dependency)

#### Success Criteria

- [ ] All repository interfaces implemented
- [ ] Connection pooling works
- [ ] Full-text search with FULLTEXT works
- [ ] All CRUD operations work
- [ ] Transactions work correctly
- [ ] UTF8MB4 encoding handles all Unicode
- [ ] Integration tests pass
- [ ] Performance meets targets

---

### T053: Create MySQL Migration System

**Status:** COMPLETED
**Priority:** P1  
**Estimated Effort:** 1.5 days

#### Description

Create database migration system for MySQL including schema creation, indexes, constraints, and initial data. Handle MySQL-specific syntax and features.

#### Technical Details

- Create SQL schema for MySQL
- Define all tables with InnoDB engine
- Use LONGTEXT for large text fields
- Use LONGBLOB for binary data
- Set CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
- Create indexes (B-tree, FULLTEXT)
- Set up foreign key constraints with CASCADE
- Create FULLTEXT indexes for search
- Implement migration versioning
- Support up/down migrations
- Create initial admin user
- Create default mailboxes

#### Files to Touch

- `internal/repository/mysql/migrations.go` (new)
- `internal/repository/mysql/migrations/001_initial_schema.sql` (new)
- `internal/repository/mysql/migrations/002_indexes.sql` (new)
- `internal/repository/mysql/migrations/003_full_text_search.sql` (new)
- `internal/repository/mysql/seed.go` (new)

#### Dependencies

- T052 (MySQL repository)

#### Success Criteria

- [ ] Schema creates all tables
- [ ] InnoDB engine used
- [ ] UTF8MB4 encoding set
- [ ] Indexes created correctly
- [ ] FULLTEXT indexes work
- [ ] Foreign keys enforced
- [ ] Migrations versioned
- [ ] Migrations idempotent
- [ ] Admin user created
- [ ] Default mailboxes created

---

### T054: Implement MongoDB Repository

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 3 days

#### Description

Implement complete MongoDB repository using document-oriented approach. Map domain models to MongoDB collections, implement queries with MongoDB query language, and leverage MongoDB-specific features like text search and aggregation.

#### Technical Details

- Use `go.mongodb.org/mongo-driver` for MongoDB
- Implement connection management with context
- Create all repository interfaces
- Map domain structs to BSON documents
- Use embedded documents for addresses
- Implement MongoDB text search indexes
- Create compound indexes for performance
- Use aggregation pipeline for complex queries
- Implement pagination with skip/limit
- Handle ObjectID for primary keys
- Use transactions for multi-document operations (MongoDB 4.0+)
- Implement proper error handling

#### Files to Touch

- `internal/repository/mongodb/mongodb.go` (new)
- `internal/repository/mongodb/connection.go` (new)
- `internal/repository/mongodb/user.go` (new)
- `internal/repository/mongodb/mailbox.go` (new)
- `internal/repository/mongodb/message.go` (new)
- `internal/repository/mongodb/attachment.go` (new)
- `internal/repository/mongodb/webhook.go` (new)
- `internal/repository/mongodb/settings.go` (new)
- `internal/repository/mongodb/search.go` (new)

#### Dependencies

- T008 (repository interfaces)
- T009 (SQLite reference implementation)
- T006 (MongoDB driver dependency)

#### Success Criteria

- [ ] All repository interfaces implemented
- [ ] Connection established successfully
- [ ] Text search indexes work
- [ ] All CRUD operations work
- [ ] Queries use appropriate indexes
- [ ] Transactions work (if MongoDB 4.0+)
- [ ] Integration tests pass
- [ ] Performance meets targets

---

### T055: Create MongoDB Indexes and Initial Data

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1 day

#### Description

Create index definitions for MongoDB collections, implement initial data seeding, and set up database initialization logic. MongoDB doesn't use migrations like SQL databases, so use programmatic index creation.

#### Technical Details

- Create index definitions for all collections
- Create unique indexes (email, username)
- Create text indexes for full-text search
- Create compound indexes for common queries
- Implement index creation on startup
- Create initial admin user if not exists
- Create default mailboxes for new users
- Handle index creation errors gracefully
- Log index creation status
- Support index version tracking

#### Files to Touch

- `internal/repository/mongodb/indexes.go` (new)
- `internal/repository/mongodb/seed.go` (new)
- `internal/repository/mongodb/init.go` (new)

#### Dependencies

- T054 (MongoDB repository)

#### Success Criteria

- [ ] Indexes created on startup
- [ ] Text indexes enable search
- [ ] Unique indexes prevent duplicates
- [ ] Compound indexes improve query performance
- [ ] Admin user created if not exists
- [ ] Default mailboxes created
- [ ] Index creation logged
- [ ] Errors handled gracefully

---

### T056: Create Database Integration Tests

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 2 days

#### Description

Create comprehensive integration tests that run against all database backends. Ensure consistent behavior across SQLite, PostgreSQL, MySQL, and MongoDB. Use Docker containers for test databases.

#### Technical Details

- Create test suite that runs against all databases
- Use Docker containers for PostgreSQL, MySQL, MongoDB
- Use in-memory SQLite for fast tests
- Test all repository operations
- Test transactions and rollbacks
- Test concurrent operations
- Test error handling
- Test migration systems
- Test search functionality
- Test performance characteristics
- Use table-driven tests for consistency
- Clean up test data after each test

#### Files to Touch

- `internal/repository/integration_test.go` (new)
- `internal/repository/testdata/docker-compose.yml` (new)
- `internal/repository/testhelpers/helpers.go` (new)

#### Dependencies

- T050-T055 (all database implementations)

#### Success Criteria

- [ ] Tests run against all databases
- [ ] All databases pass same tests
- [ ] Docker containers start automatically
- [ ] Tests clean up after themselves
- [ ] Performance tests measure each database
- [ ] Concurrent operations safe
- [ ] Search works consistently
- [ ] Test coverage > 70%

---

### T057: Update Repository Factory and Configuration

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 0.5 days

#### Description

Update the repository factory to instantiate PostgreSQL, MySQL, and MongoDB repositories based on configuration. Update configuration examples and documentation for all database backends.

#### Technical Details

- Update factory to handle postgres, mysql, mongodb drivers
- Validate configuration for each driver
- Return appropriate repository implementation
- Handle driver-specific connection strings
- Update example configuration files
- Document connection string formats
- Document database-specific considerations
- Provide Docker Compose examples for each database

#### Files to Touch

- `internal/repository/factory.go` (update)
- `configs/yunt.example.yaml` (update)
- `configs/yunt.postgres.yaml` (new)
- `configs/yunt.mysql.yaml` (new)
- `configs/yunt.mongodb.yaml` (new)

#### Dependencies

- T050-T055 (all repository implementations)
- T011 (existing factory)

#### Success Criteria

- [ ] Factory creates correct repository
- [ ] All drivers supported
- [ ] Configuration validated
- [ ] Example configs provided
- [ ] Documentation complete
- [ ] Docker Compose examples work

---

### T058: Performance Testing and Optimization

**Status:** COMPLETED
**Priority:** P2  
**Estimated Effort:** 1.5 days

#### Description

Conduct performance testing across all database backends. Identify bottlenecks, optimize queries, add indexes where needed, and document performance characteristics of each database.

#### Technical Details

- Create benchmark tests for common operations
- Test insert performance (single, bulk)
- Test query performance (list, search, filter)
- Test update performance
- Test delete performance
- Measure memory usage
- Measure connection overhead
- Profile query execution plans
- Add missing indexes
- Optimize slow queries
- Document performance findings
- Create performance comparison chart

#### Files to Touch

- `internal/repository/benchmark_test.go` (new)
- `docs/performance.md` (new)

#### Dependencies

- T050-T055 (all repository implementations)

#### Success Criteria

- [ ] Benchmark tests for all operations
- [ ] Performance meets targets for each database
- [ ] Bottlenecks identified and optimized
- [ ] Query plans analyzed
- [ ] Missing indexes added
- [ ] Performance documented
- [ ] Comparison chart created

---

## Performance Targets

| Operation         | SQLite  | PostgreSQL | MySQL   | MongoDB |
|-------------------|---------|------------|---------|---------|
| Insert (single)   | < 5ms   | < 10ms     | < 10ms  | < 10ms  |
| Insert (bulk 100) | < 50ms  | < 100ms    | < 100ms | < 100ms |
| Query (list 50)   | < 10ms  | < 20ms     | < 20ms  | < 20ms  |
| Search (10k msgs) | < 100ms | < 50ms     | < 50ms  | < 100ms |
| Update (single)   | < 5ms   | < 10ms     | < 10ms  | < 10ms  |
| Delete (single)   | < 5ms   | < 10ms     | < 10ms  | < 10ms  |

## Risk Assessment

| Risk                              | Probability | Impact | Mitigation                                   |
|-----------------------------------|-------------|--------|----------------------------------------------|
| Database-specific bugs            | Medium      | Medium | Comprehensive integration tests              |
| Performance differences           | High        | Medium | Benchmarking, optimization, documentation    |
| Migration complexity              | Medium      | Medium | Migration tools, clear documentation         |
| MongoDB transaction limitations   | Low         | Medium | Document minimum MongoDB version (4.0+)      |
| Connection pool tuning            | Medium      | Low    | Provide sensible defaults, document tuning   |

## Notes

- SQLite is best for development and small deployments
- PostgreSQL recommended for production deployments
- MySQL provides compatibility with existing infrastructure
- MongoDB offers flexible schema and horizontal scaling
- Full-text search performance varies significantly between databases
- Consider documenting data migration paths between databases
- Connection pool settings critical for performance
- Some features may require minimum database versions
- PostgreSQL offers best full-text search capabilities
- MongoDB requires replica set for transactions
- Consider implementing database health checks
- Document backup and restore procedures for each database
- Consider implementing read replicas support (future)
