// Package repository provides comprehensive benchmarks for all database operations.
// These benchmarks test CRUD operations, search, bulk operations, and complex queries
// across all supported database backends (SQLite, PostgreSQL, MySQL, MongoDB).
//
// Run benchmarks:
//
//	go test -bench=. -benchmem ./internal/repository/sqlite/...
//
// Run specific benchmark:
//
//	go test -bench=BenchmarkUserCreate -benchmem ./internal/repository/sqlite/...
//
// Compare backends:
//
//	go test -bench=. -benchmem -count=5 ./internal/repository/sqlite/... | tee results.txt
package repository

// This file documents the benchmark test structure.
// Actual benchmarks are implemented in each database backend package:
//
// - internal/repository/sqlite/benchmark_test.go
// - internal/repository/postgres/benchmark_test.go (when implemented)
// - internal/repository/mysql/benchmark_test.go (when implemented)
// - internal/repository/mongodb/benchmark_test.go (when implemented)
//
// Each backend package contains benchmarks for:
//
// User Operations:
//   - BenchmarkUserCreate
//   - BenchmarkUserGetByID
//   - BenchmarkUserGetByUsername
//   - BenchmarkUserGetByEmail
//   - BenchmarkUserUpdate
//   - BenchmarkUserList
//   - BenchmarkUserSearch
//   - BenchmarkUserCount
//   - BenchmarkUserExists
//
// Mailbox Operations:
//   - BenchmarkMailboxCreate
//   - BenchmarkMailboxGetByID
//   - BenchmarkMailboxGetByAddress
//   - BenchmarkMailboxListByUser
//   - BenchmarkMailboxUpdateStats
//
// Message Operations:
//   - BenchmarkMessageCreate
//   - BenchmarkMessageGetByID
//   - BenchmarkMessageGetByMessageID
//   - BenchmarkMessageListByMailbox
//   - BenchmarkMessageSearch
//   - BenchmarkMessageMarkAsRead
//   - BenchmarkMessageStar
//   - BenchmarkMessageCountByMailbox
//   - BenchmarkMessageBulkMarkAsRead
//
// Attachment Operations:
//   - BenchmarkAttachmentCreate
//   - BenchmarkAttachmentGetByID
//   - BenchmarkAttachmentListByMessage
//
// Webhook Operations:
//   - BenchmarkWebhookCreate
//   - BenchmarkWebhookListByUser
//   - BenchmarkWebhookListActiveByEvent
//   - BenchmarkWebhookRecordSuccess
//
// Transaction Operations:
//   - BenchmarkTransaction
//   - BenchmarkTransactionWithMultipleOps
//
// Concurrent Access:
//   - BenchmarkConcurrentReads
//   - BenchmarkConcurrentWrites
//   - BenchmarkConcurrentMixedOps
//
// Settings Operations:
//   - BenchmarkSettingsGet
//   - BenchmarkSettingsSave
//
// Complex Queries:
//   - BenchmarkComplexMessageFilter
//   - BenchmarkComplexUserFilter
//
// Aggregations:
//   - BenchmarkUserCountByRole
//   - BenchmarkMailboxGetTotalStats
