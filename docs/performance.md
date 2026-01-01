# Yunt Performance Guide

This document provides comprehensive information about Yunt's database performance characteristics, benchmarking methodology, optimization strategies, and recommendations for each supported database backend.

## Table of Contents

1. [Overview](#overview)
2. [Running Benchmarks](#running-benchmarks)
3. [Database Backend Comparison](#database-backend-comparison)
4. [Operation Performance](#operation-performance)
5. [Index Strategy](#index-strategy)
6. [Query Optimization](#query-optimization)
7. [Scaling Recommendations](#scaling-recommendations)
8. [Monitoring and Profiling](#monitoring-and-profiling)

## Overview

Yunt supports four database backends, each with different performance characteristics:

| Backend    | Best For                              | Typical Latency | Concurrency Support |
|------------|---------------------------------------|-----------------|---------------------|
| SQLite     | Development, single-server, low load  | <1ms            | Limited (WAL mode)  |
| PostgreSQL | Production, high concurrency          | 1-5ms           | Excellent           |
| MySQL      | Production, read-heavy workloads      | 1-5ms           | Good                |
| MongoDB    | Flexible schema, document-heavy       | 1-10ms          | Excellent           |

## Running Benchmarks

### Prerequisites

For SQLite benchmarks, CGO must be enabled with a C compiler available:

```bash
# Linux/macOS
export CGO_ENABLED=1

# Windows (with MinGW or similar)
set CGO_ENABLED=1
```

### Running All Benchmarks

```bash
# Run all repository benchmarks
go test -bench=. -benchmem ./internal/repository/sqlite/...

# Run with longer benchmark time for more accurate results
go test -bench=. -benchmem -benchtime=3s ./internal/repository/sqlite/...

# Run specific benchmark
go test -bench=BenchmarkUserCreate -benchmem ./internal/repository/sqlite/...

# Generate comparison data
go test -bench=. -benchmem -count=5 ./internal/repository/sqlite/... | tee results.txt
```

### Benchmark Categories

The benchmark suite covers the following operation categories:

| Category              | Benchmarks                                                    |
|-----------------------|---------------------------------------------------------------|
| User Operations       | Create, GetByID, GetByUsername, GetByEmail, Update, List     |
| Mailbox Operations    | Create, GetByID, GetByAddress, ListByUser, UpdateStats        |
| Message Operations    | Create, GetByID, ListByMailbox, Search, MarkAsRead, BulkOps  |
| Attachment Operations | Create, GetByID, ListByMessage                                |
| Webhook Operations    | Create, ListByUser, ListActiveByEvent, RecordSuccess          |
| Transactions          | Simple, MultipleOps                                           |
| Concurrent Access     | ConcurrentReads, ConcurrentWrites, MixedOps                   |
| Complex Queries       | ComplexMessageFilter, ComplexUserFilter                       |
| Aggregations          | CountByRole, GetTotalStats                                    |

## Database Backend Comparison

### Performance Characteristics

#### SQLite

**Strengths:**
- Zero-latency for local operations (no network overhead)
- Excellent for read-heavy workloads
- Simple deployment with no external dependencies
- Very fast for small to medium datasets (<100K messages)

**Weaknesses:**
- Single-writer limitation (writes are serialized)
- Performance degrades with concurrent writes
- Not suitable for multi-server deployments

**Optimal Settings:**
```yaml
database:
  driver: sqlite
  dsn: "./data/yunt.db"
  maxOpenConns: 1
  maxIdleConns: 1
  pragmas:
    journal_mode: WAL
    synchronous: NORMAL
    cache_size: -64000  # 64MB cache
    temp_store: MEMORY
```

#### PostgreSQL

**Strengths:**
- Excellent concurrent read/write performance
- Advanced indexing (GIN, GiST for full-text search)
- MVCC for consistent reads during writes
- Connection pooling support

**Weaknesses:**
- Requires external server
- Higher memory usage
- Configuration complexity

**Optimal Settings:**
```yaml
database:
  driver: postgres
  host: localhost
  port: 5432
  database: yunt
  maxOpenConns: 25
  maxIdleConns: 5
  connMaxLifetime: 5m
```

**PostgreSQL Configuration:**
```sql
-- Recommended postgresql.conf settings for Yunt
shared_buffers = 256MB
effective_cache_size = 768MB
work_mem = 16MB
maintenance_work_mem = 128MB
random_page_cost = 1.1
effective_io_concurrency = 200
```

#### MySQL

**Strengths:**
- Good read performance with proper indexing
- Wide hosting support
- Familiar to many developers

**Weaknesses:**
- Write performance can be slower than PostgreSQL
- Less sophisticated query optimizer
- Full-text search requires InnoDB with specific setup

**Optimal Settings:**
```yaml
database:
  driver: mysql
  host: localhost
  port: 3306
  database: yunt
  maxOpenConns: 25
  maxIdleConns: 5
  parseTime: true
```

#### MongoDB

**Strengths:**
- Flexible document storage
- Horizontal scaling with sharding
- Rich query capabilities on nested documents

**Weaknesses:**
- Higher storage overhead
- Potential for inconsistent data without schemas
- More complex backup and maintenance

**Optimal Settings:**
```yaml
database:
  driver: mongodb
  dsn: "mongodb://localhost:27017"
  database: yunt
  maxPoolSize: 25
  minPoolSize: 5
```

### Expected Performance Comparison

The following table shows expected operation latency (median) for each backend:

| Operation              | SQLite    | PostgreSQL | MySQL     | MongoDB   |
|------------------------|-----------|------------|-----------|-----------|
| User Create            | <0.5ms    | 1-2ms      | 1-2ms     | 1-3ms     |
| User GetByID           | <0.1ms    | 0.5-1ms    | 0.5-1ms   | 1-2ms     |
| User GetByUsername     | <0.1ms    | 0.5-1ms    | 0.5-1ms   | 1-2ms     |
| User List (20 items)   | <0.5ms    | 1-2ms      | 1-2ms     | 2-3ms     |
| User Search            | 1-5ms     | 1-3ms      | 2-5ms     | 2-5ms     |
| Message Create         | <0.5ms    | 1-2ms      | 1-2ms     | 2-4ms     |
| Message GetByID        | <0.1ms    | 0.5-1ms    | 0.5-1ms   | 1-2ms     |
| Message List (20 items)| <0.5ms    | 1-2ms      | 1-2ms     | 2-4ms     |
| Message Search         | 2-10ms    | 1-5ms      | 3-10ms    | 2-8ms     |
| Bulk Mark Read (50)    | 1-5ms     | 2-5ms      | 2-5ms     | 5-10ms    |
| Transaction (3 ops)    | <1ms      | 2-5ms      | 2-5ms     | 5-10ms    |

*Note: Latencies measured with local database, production deployments may vary.*

## Operation Performance

### Write Operations

Write operations are critical for mail server performance. Key optimizations:

1. **Batch Inserts**: Use bulk operations when creating multiple records
2. **Prepared Statements**: All queries use prepared statements for efficiency
3. **Transaction Batching**: Group related writes in transactions

```go
// Example: Efficient bulk message creation
err := repo.Transaction(ctx, func(tx repository.Repository) error {
    for _, msg := range messages {
        if err := tx.Messages().Create(ctx, msg); err != nil {
            return err
        }
    }
    return nil
})
```

### Read Operations

Read operations are optimized through:

1. **Indexed Lookups**: Primary keys and common query fields are indexed
2. **Pagination**: List operations use efficient OFFSET/LIMIT or cursor-based pagination
3. **Selective Loading**: Only required fields are loaded

### Search Operations

Full-text search performance varies by backend:

| Backend    | Search Technology      | Index Type     | Typical Performance |
|------------|------------------------|----------------|---------------------|
| SQLite     | LIKE with indexes      | B-tree         | 2-10ms (1K records) |
| PostgreSQL | Full-text search       | GIN/GiST       | 1-5ms (10K records) |
| MySQL      | FULLTEXT indexes       | InnoDB FT      | 3-10ms (10K records)|
| MongoDB    | Text indexes           | Text Index     | 2-8ms (10K records) |

## Index Strategy

### SQLite Indexes

The SQLite schema includes comprehensive indexes:

```sql
-- User indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_users_created_at ON users(created_at);

-- Message indexes
CREATE INDEX idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_from_address ON messages(from_address);
CREATE INDEX idx_messages_status ON messages(status);
CREATE INDEX idx_messages_is_starred ON messages(is_starred);
CREATE INDEX idx_messages_received_at ON messages(received_at);
CREATE INDEX idx_messages_subject ON messages(subject);

-- Mailbox indexes
CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);
CREATE INDEX idx_mailboxes_address ON mailboxes(address);
CREATE INDEX idx_mailboxes_is_catch_all ON mailboxes(is_catch_all);

-- Webhook indexes
CREATE INDEX idx_webhooks_user_id ON webhooks(user_id);
CREATE INDEX idx_webhooks_status ON webhooks(status);
CREATE INDEX idx_webhook_deliveries_webhook_id ON webhook_deliveries(webhook_id);
```

### PostgreSQL Indexes

PostgreSQL benefits from additional index types:

```sql
-- GIN index for full-text search
CREATE INDEX idx_messages_search ON messages
USING GIN (to_tsvector('english', subject || ' ' || COALESCE(text_body, '')));

-- Partial indexes for common queries
CREATE INDEX idx_messages_unread ON messages (mailbox_id)
WHERE status = 'unread';

CREATE INDEX idx_messages_starred ON messages (mailbox_id)
WHERE is_starred = true;

-- Composite indexes for common filter combinations
CREATE INDEX idx_messages_mailbox_received
ON messages (mailbox_id, received_at DESC);
```

### Index Maintenance

Regular index maintenance improves performance:

```sql
-- SQLite: Analyze and optimize
ANALYZE;
VACUUM;

-- PostgreSQL: Reindex and analyze
REINDEX DATABASE yunt;
ANALYZE;

-- MySQL: Optimize tables
OPTIMIZE TABLE messages, mailboxes, users;
ANALYZE TABLE messages, mailboxes, users;
```

## Query Optimization

### Pagination Best Practices

For large result sets, use keyset pagination instead of OFFSET:

```sql
-- Instead of:
SELECT * FROM messages WHERE mailbox_id = ? ORDER BY received_at DESC OFFSET 1000 LIMIT 20;

-- Use keyset pagination:
SELECT * FROM messages
WHERE mailbox_id = ? AND received_at < ?
ORDER BY received_at DESC
LIMIT 20;
```

### Query Plan Analysis

Analyze slow queries using database-specific tools:

```sql
-- SQLite
EXPLAIN QUERY PLAN SELECT * FROM messages WHERE mailbox_id = 'xyz';

-- PostgreSQL
EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM messages WHERE mailbox_id = 'xyz';

-- MySQL
EXPLAIN ANALYZE SELECT * FROM messages WHERE mailbox_id = 'xyz';
```

### Common Query Patterns

The repository layer optimizes these common patterns:

| Pattern                  | Optimization                              |
|--------------------------|-------------------------------------------|
| Get by ID                | Primary key lookup                        |
| Get by unique field      | Unique index lookup                       |
| List with pagination     | Index-covered query with LIMIT            |
| Filter by status         | Partial index (PostgreSQL) or index scan  |
| Search by text           | Full-text index where available           |
| Count operations         | Index-only scan where possible            |
| Date range queries       | B-tree index on timestamp columns         |

## Scaling Recommendations

### Small Deployment (< 10K messages/day)

- **Recommended Backend**: SQLite
- **Configuration**: Single server with WAL mode
- **Memory**: 512MB+ for database cache

### Medium Deployment (10K-100K messages/day)

- **Recommended Backend**: PostgreSQL or MySQL
- **Configuration**: Dedicated database server
- **Memory**: 2GB+ with proper connection pooling
- **Connections**: 25-50 max connections

### Large Deployment (> 100K messages/day)

- **Recommended Backend**: PostgreSQL with read replicas
- **Configuration**:
  - Primary server for writes
  - Read replicas for search and list operations
  - Connection pooler (PgBouncer)
- **Memory**: 8GB+ per database server
- **Connections**: Pool with 100+ connections

### MongoDB Horizontal Scaling

For MongoDB deployments requiring horizontal scaling:

```yaml
# Sharding configuration
database:
  driver: mongodb
  dsn: "mongodb://mongos:27017"
  database: yunt

# Shard key recommendations
# messages: { mailbox_id: 1, received_at: 1 }
# users: { _id: "hashed" }
```

## Monitoring and Profiling

### Key Metrics to Monitor

| Metric                     | Target        | Alert Threshold |
|----------------------------|---------------|-----------------|
| Query latency (p95)        | < 10ms        | > 100ms         |
| Query latency (p99)        | < 50ms        | > 500ms         |
| Connection pool usage      | < 80%         | > 90%           |
| Slow queries per minute    | 0             | > 10            |
| Lock wait time             | < 1ms         | > 100ms         |
| Cache hit ratio            | > 95%         | < 80%           |

### Enabling Query Logging

For debugging slow queries:

```yaml
# SQLite
database:
  dsn: "./data/yunt.db?_trace=stderr"

# PostgreSQL (in postgresql.conf)
log_min_duration_statement = 100  # Log queries over 100ms

# MySQL (in my.cnf)
slow_query_log = 1
long_query_time = 0.1
```

### Profiling Tools

| Backend    | Profiling Tool               |
|------------|------------------------------|
| SQLite     | sqlite3 CLI with `.timer on` |
| PostgreSQL | pg_stat_statements, EXPLAIN  |
| MySQL      | Performance Schema           |
| MongoDB    | mongod profiler              |

### Health Check Endpoints

The Yunt API provides database health information:

```bash
# Check database health
curl http://localhost:8025/api/v1/health

# Response includes database status
{
  "status": "healthy",
  "database": {
    "driver": "sqlite",
    "connected": true,
    "latency_ms": 0.5
  }
}
```

## Benchmark Results Template

When running benchmarks, document results using this template:

```
# Yunt Benchmark Results
# Date: YYYY-MM-DD
# Hardware: [CPU, Memory, Storage type]
# Database: [Backend and version]
# Dataset: [Number of users, mailboxes, messages]

BenchmarkUserCreate-8             10000      0.45 ms/op       1024 B/op      15 allocs/op
BenchmarkUserGetByID-8            50000      0.08 ms/op        512 B/op       8 allocs/op
BenchmarkMessageCreate-8          10000      0.52 ms/op       2048 B/op      22 allocs/op
BenchmarkMessageListByMailbox-8   20000      0.35 ms/op       4096 B/op      45 allocs/op
BenchmarkMessageSearch-8           5000      2.10 ms/op       8192 B/op      85 allocs/op
```

## Troubleshooting Performance Issues

### High Latency

1. Check for missing indexes: `EXPLAIN` queries
2. Review connection pool settings
3. Monitor lock contention
4. Check disk I/O performance

### Memory Issues

1. Reduce connection pool size
2. Lower cache sizes for SQLite
3. Review query result set sizes
4. Enable query pagination

### Connection Exhaustion

1. Increase `maxOpenConns` gradually
2. Implement connection pooling (PgBouncer, ProxySQL)
3. Review connection lifecycle settings
4. Check for connection leaks

## References

- [SQLite Performance Documentation](https://www.sqlite.org/np1queryprob.html)
- [PostgreSQL Performance Tips](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [MySQL Optimization Guide](https://dev.mysql.com/doc/refman/8.0/en/optimization.html)
- [MongoDB Performance Best Practices](https://www.mongodb.com/docs/manual/administration/analyzing-mongodb-performance/)
