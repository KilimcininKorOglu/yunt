package mongodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"yunt/internal/domain"
)

// DBSessionStore implements service.SessionStore using MongoDB.
type DBSessionStore struct {
	repo *Repository
}

type sessionDocument struct {
	ID               string    `bson:"_id"`
	UserID           string    `bson:"userId"`
	RefreshTokenHash string    `bson:"refreshTokenHash"`
	UserAgent        string    `bson:"userAgent"`
	IPAddress        string    `bson:"ipAddress"`
	IsRevoked        bool      `bson:"isRevoked"`
	CreatedAt        time.Time `bson:"createdAt"`
	ExpiresAt        time.Time `bson:"expiresAt"`
	LastUsedAt       time.Time `bson:"lastUsedAt"`
}

// NewDBSessionStore creates a new database-backed session store.
func NewDBSessionStore(repo *Repository) *DBSessionStore {
	return &DBSessionStore{repo: repo}
}

func (s *DBSessionStore) collection() mongoCollection {
	return s.repo.collection("sessions")
}

func (s *DBSessionStore) Create(_ context.Context, session *domain.Session) error {
	doc := &sessionDocument{
		ID:               session.ID,
		UserID:           string(session.UserID),
		RefreshTokenHash: session.RefreshTokenHash,
		UserAgent:        session.UserAgent,
		IPAddress:        session.IPAddress,
		IsRevoked:        session.IsRevoked,
		CreatedAt:        session.CreatedAt.Time,
		ExpiresAt:        session.ExpiresAt.Time,
		LastUsedAt:       session.LastUsedAt.Time,
	}

	_, err := s.collection().InsertOne(context.Background(), doc)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Get(_ context.Context, id string) (*domain.Session, error) {
	var doc sessionDocument
	err := s.collection().FindOne(context.Background(), bson.M{"_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return docToSession(&doc), nil
}

func (s *DBSessionStore) Update(_ context.Context, session *domain.Session) error {
	filter := bson.M{"_id": session.ID}
	update := bson.M{"$set": bson.M{
		"refreshTokenHash": session.RefreshTokenHash,
		"userAgent":        session.UserAgent,
		"ipAddress":        session.IPAddress,
		"isRevoked":        session.IsRevoked,
		"lastUsedAt":       session.LastUsedAt.Time,
	}}

	_, err := s.collection().UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (s *DBSessionStore) Delete(_ context.Context, id string) error {
	_, err := s.collection().DeleteOne(context.Background(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

func (s *DBSessionStore) DeleteByUserID(_ context.Context, userID domain.ID) error {
	_, err := s.collection().DeleteMany(context.Background(), bson.M{"userId": string(userID)})
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (s *DBSessionStore) Touch(_ context.Context, id string) error {
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"lastUsedAt": time.Now().UTC()}}

	_, err := s.collection().UpdateOne(context.Background(), filter, update)
	if err != nil {
		return fmt.Errorf("failed to touch session: %w", err)
	}
	return nil
}

// EnsureIndexes creates indexes for the sessions collection.
func (s *DBSessionStore) EnsureIndexes(ctx context.Context) error {
	indexes := []mongo.IndexModel{
		{Keys: bson.M{"userId": 1}},
		{Keys: bson.M{"expiresAt": 1}, Options: options.Index().SetExpireAfterSeconds(0)},
	}

	_, err := s.repo.pool.Collection("sessions").Indexes().CreateMany(ctx, indexes)
	return err
}

func docToSession(doc *sessionDocument) *domain.Session {
	return &domain.Session{
		ID:               doc.ID,
		UserID:           domain.ID(doc.UserID),
		RefreshTokenHash: doc.RefreshTokenHash,
		UserAgent:        doc.UserAgent,
		IPAddress:        doc.IPAddress,
		IsRevoked:        doc.IsRevoked,
		CreatedAt:        domain.Timestamp{Time: doc.CreatedAt},
		ExpiresAt:        domain.Timestamp{Time: doc.ExpiresAt},
		LastUsedAt:       domain.Timestamp{Time: doc.LastUsedAt},
	}
}
