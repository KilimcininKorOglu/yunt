package mysql

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"yunt/internal/metrics"
)

type metricsDB struct {
	inner sqlxDB
}

func (m *metricsDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := m.inner.ExecContext(ctx, query, args...)
	m.record(query, start)
	return result, err
}

func (m *metricsDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := m.inner.QueryContext(ctx, query, args...)
	m.record(query, start)
	return rows, err
}

func (m *metricsDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := m.inner.QueryRowContext(ctx, query, args...)
	m.record(query, start)
	return row
}

func (m *metricsDB) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := m.inner.GetContext(ctx, dest, query, args...)
	m.record(query, start)
	return err
}

func (m *metricsDB) SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	start := time.Now()
	err := m.inner.SelectContext(ctx, dest, query, args...)
	m.record(query, start)
	return err
}

func (m *metricsDB) NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := m.inner.NamedExecContext(ctx, query, arg)
	m.record(query, start)
	return result, err
}

func (m *metricsDB) Rebind(query string) string {
	return m.inner.Rebind(query)
}

func (m *metricsDB) record(query string, start time.Time) {
	op := sqlOperation(query)
	elapsed := time.Since(start).Seconds()
	metrics.DBQueriesTotal.WithLabelValues(op).Inc()
	metrics.DBQueryDuration.WithLabelValues(op).Observe(elapsed)
}

func sqlOperation(query string) string {
	q := strings.TrimSpace(query)
	if len(q) < 6 {
		return "other"
	}
	switch strings.ToLower(q[:6]) {
	case "select":
		return "select"
	case "insert":
		return "insert"
	case "update":
		return "update"
	case "delete":
		return "delete"
	default:
		return "other"
	}
}
