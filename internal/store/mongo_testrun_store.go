package store

import (
	"context"
	"fmt"
	"time"

	"Load-manager-cli/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	testRunsCollection = "test_runs"
)

// MongoTestRunStore implements TestRunRepository using MongoDB
type MongoTestRunStore struct {
	collection *mongo.Collection
}

// NewMongoTestRunStore creates a new MongoDB test run store
func NewMongoTestRunStore(db *mongo.Database) (*MongoTestRunStore, error) {
	collection := db.Collection(testRunsCollection)
	
	store := &MongoTestRunStore{
		collection: collection,
	}
	
	if err := store.createIndexes(); err != nil {
		return nil, err
	}
	
	return store, nil
}

// createIndexes creates optimized indexes for test runs
func (s *MongoTestRunStore) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("id_unique_idx"),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "envId", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("tenant_env_status_idx"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_idx"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("created_at_idx"),
		},
		{
			Keys: bson.D{
				{Key: "tenantId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("tenant_created_idx"),
		},
	}
	
	_, err := s.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}
	
	return nil
}

// Create stores a new test run
func (s *MongoTestRunStore) Create(ctx context.Context, testRun *domain.TestRun) error {
	_, err := s.collection.InsertOne(ctx, testRun)
	if err != nil {
		return fmt.Errorf("failed to create test run: %w", err)
	}
	return nil
}

// Update updates an existing test run
func (s *MongoTestRunStore) Update(ctx context.Context, testRun *domain.TestRun) error {
	filter := bson.M{"id": testRun.ID}
	update := bson.M{"$set": testRun}
	
	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update test run: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("test run not found: %s", testRun.ID)
	}
	
	return nil
}

// GetByID retrieves a test run by ID
func (s *MongoTestRunStore) GetByID(ctx context.Context, id string) (*domain.TestRun, error) {
	var testRun domain.TestRun
	
	err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(&testRun)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("test run not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get test run: %w", err)
	}
	
	return &testRun, nil
}

// List retrieves test runs with optional filters
func (s *MongoTestRunStore) List(ctx context.Context, tenantID, envID, status string) ([]*domain.TestRun, error) {
	filter := bson.M{}
	
	if tenantID != "" {
		filter["tenantId"] = tenantID
	}
	if envID != "" {
		filter["envId"] = envID
	}
	if status != "" {
		filter["status"] = status
	}
	
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	
	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list test runs: %w", err)
	}
	defer cursor.Close(ctx)
	
	var testRuns []*domain.TestRun
	if err := cursor.All(ctx, &testRuns); err != nil {
		return nil, fmt.Errorf("failed to decode test runs: %w", err)
	}
	
	return testRuns, nil
}

// Delete deletes a test run by ID
func (s *MongoTestRunStore) Delete(ctx context.Context, id string) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete test run: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("test run not found: %s", id)
	}
	
	return nil
}
