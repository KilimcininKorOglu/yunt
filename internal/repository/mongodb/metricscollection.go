package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/metrics"
)

type mongoCollection interface {
	FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult
	Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error)
	InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error)
	DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error)
	CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error)
	Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error)
	ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error)
	FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult
}

type metricsCollection struct {
	inner mongoCollection
}

func (m *metricsCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	start := time.Now()
	result := m.inner.FindOne(ctx, filter, opts...)
	m.record("find_one", start)
	return result
}

func (m *metricsCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	start := time.Now()
	cur, err := m.inner.Find(ctx, filter, opts...)
	m.record("find", start)
	return cur, err
}

func (m *metricsCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	start := time.Now()
	result, err := m.inner.InsertOne(ctx, document, opts...)
	m.record("insert", start)
	return result, err
}

func (m *metricsCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := m.inner.UpdateOne(ctx, filter, update, opts...)
	m.record("update", start)
	return result, err
}

func (m *metricsCollection) UpdateMany(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := m.inner.UpdateMany(ctx, filter, update, opts...)
	m.record("update", start)
	return result, err
}

func (m *metricsCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	start := time.Now()
	result, err := m.inner.DeleteOne(ctx, filter, opts...)
	m.record("delete", start)
	return result, err
}

func (m *metricsCollection) DeleteMany(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	start := time.Now()
	result, err := m.inner.DeleteMany(ctx, filter, opts...)
	m.record("delete", start)
	return result, err
}

func (m *metricsCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	start := time.Now()
	count, err := m.inner.CountDocuments(ctx, filter, opts...)
	m.record("count", start)
	return count, err
}

func (m *metricsCollection) Aggregate(ctx context.Context, pipeline interface{}, opts ...*options.AggregateOptions) (*mongo.Cursor, error) {
	start := time.Now()
	cur, err := m.inner.Aggregate(ctx, pipeline, opts...)
	m.record("aggregate", start)
	return cur, err
}

func (m *metricsCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	start := time.Now()
	result, err := m.inner.ReplaceOne(ctx, filter, replacement, opts...)
	m.record("replace", start)
	return result, err
}

func (m *metricsCollection) FindOneAndUpdate(ctx context.Context, filter interface{}, update interface{}, opts ...*options.FindOneAndUpdateOptions) *mongo.SingleResult {
	start := time.Now()
	result := m.inner.FindOneAndUpdate(ctx, filter, update, opts...)
	m.record("find_and_update", start)
	return result
}

func (m *metricsCollection) record(op string, start time.Time) {
	elapsed := time.Since(start).Seconds()
	metrics.DBQueriesTotal.WithLabelValues(op).Inc()
	metrics.DBQueryDuration.WithLabelValues(op).Observe(elapsed)
}
