package store

import (
	"Load-manager-cli/internal/domain"
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ScriptRevisionRepository defines the interface for script revision storage
type ScriptRevisionRepository interface {
	Create(revision *domain.ScriptRevision) error
	Get(id string) (*domain.ScriptRevision, error)
	GetLatestByLoadTestID(loadTestID string) (*domain.ScriptRevision, error)
	ListByLoadTestID(loadTestID string, limit int) ([]*domain.ScriptRevision, error)
}

// MongoScriptRevisionStore implements ScriptRevisionRepository using MongoDB
type MongoScriptRevisionStore struct {
	collection *mongo.Collection
}

// NewMongoScriptRevisionStore creates a new MongoDB-backed script revision store
func NewMongoScriptRevisionStore(db *mongo.Database) (*MongoScriptRevisionStore, error) {
	collection := db.Collection("script_revisions")
	
	ctx := context.Background()
	
	// Create indexes
	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "loadTestId", Value: 1},
				{Key: "revisionNumber", Value: -1}, // Descending for latest first
			},
		},
		{
			Keys: bson.D{
				{Key: "loadTestId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
		},
	}
	
	_, err := collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}
	
	return &MongoScriptRevisionStore{collection: collection}, nil
}

// Create stores a new script revision
func (s *MongoScriptRevisionStore) Create(revision *domain.ScriptRevision) error {
	ctx := context.Background()
	_, err := s.collection.InsertOne(ctx, revision)
	if err != nil {
		return fmt.Errorf("failed to create script revision: %w", err)
	}
	return nil
}

// Get retrieves a script revision by ID
func (s *MongoScriptRevisionStore) Get(id string) (*domain.ScriptRevision, error) {
	ctx := context.Background()
	
	var revision domain.ScriptRevision
	err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(&revision)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("script revision not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get script revision: %w", err)
	}
	
	return &revision, nil
}

// GetLatestByLoadTestID retrieves the latest revision for a load test
func (s *MongoScriptRevisionStore) GetLatestByLoadTestID(loadTestID string) (*domain.ScriptRevision, error) {
	ctx := context.Background()
	
	opts := options.FindOne().SetSort(bson.D{{Key: "revisionNumber", Value: -1}})
	
	var revision domain.ScriptRevision
	err := s.collection.FindOne(ctx, bson.M{"loadTestId": loadTestID}, opts).Decode(&revision)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no script revisions found for load test: %s", loadTestID)
		}
		return nil, fmt.Errorf("failed to get latest script revision: %w", err)
	}
	
	return &revision, nil
}

// ListByLoadTestID retrieves revisions for a load test (most recent first)
func (s *MongoScriptRevisionStore) ListByLoadTestID(loadTestID string, limit int) ([]*domain.ScriptRevision, error) {
	ctx := context.Background()
	
	if limit <= 0 {
		limit = 10 // Default limit
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "revisionNumber", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := s.collection.Find(ctx, bson.M{"loadTestId": loadTestID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list script revisions: %w", err)
	}
	defer cursor.Close(ctx)
	
	var revisions []*domain.ScriptRevision
	if err := cursor.All(ctx, &revisions); err != nil {
		return nil, fmt.Errorf("failed to decode script revisions: %w", err)
	}
	
	return revisions, nil
}
