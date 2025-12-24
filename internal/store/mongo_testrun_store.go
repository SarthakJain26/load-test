package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"Load-manager-cli/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	loadTestsCollection    = "load_tests"
	loadTestRunsCollection = "load_test_runs"
)

// MongoLoadTestStore implements LoadTestRepository using MongoDB
type MongoLoadTestStore struct {
	collection *mongo.Collection
}

// NewMongoLoadTestStore creates a new MongoDB load test store
func NewMongoLoadTestStore(db *mongo.Database) (*MongoLoadTestStore, error) {
	collection := db.Collection(loadTestsCollection)
	
	store := &MongoLoadTestStore{
		collection: collection,
	}
	
	if err := store.createIndexes(); err != nil {
		return nil, err
	}
	
	return store, nil
}

// MongoLoadTestRunStore implements LoadTestRunRepository using MongoDB
type MongoLoadTestRunStore struct {
	collection *mongo.Collection
}

// NewMongoLoadTestRunStore creates a new MongoDB load test run store
func NewMongoLoadTestRunStore(db *mongo.Database) (*MongoLoadTestRunStore, error) {
	collection := db.Collection(loadTestRunsCollection)
	
	store := &MongoLoadTestRunStore{
		collection: collection,
	}
	
	if err := store.createIndexes(); err != nil {
		return nil, err
	}
	
	return store, nil
}

// LoadTest methods

// createIndexes creates optimized indexes for load tests
func (s *MongoLoadTestStore) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("id_unique_idx"),
		},
		{
			Keys: bson.D{
				{Key: "accountId", Value: 1},
				{Key: "orgId", Value: 1},
				{Key: "projectId", Value: 1},
			},
			Options: options.Index().SetName("account_org_project_idx"),
		},
		{
			Keys:    bson.D{{Key: "tags", Value: 1}},
			Options: options.Index().SetName("tags_idx"),
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
		return fmt.Errorf("failed to create load test indexes: %w", err)
	}
	
	return nil
}

// Create stores a new load test
func (s *MongoLoadTestStore) Create(test *domain.LoadTest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := s.collection.InsertOne(ctx, test)
	if err != nil {
		return fmt.Errorf("failed to create load test: %w", err)
	}
	return nil
}

// Get retrieves a load test by ID
func (s *MongoLoadTestStore) Get(id string) (*domain.LoadTest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var test domain.LoadTest
	err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(&test)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("load test not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get load test: %w", err)
	}
	
	return &test, nil
}

// Update updates an existing load test
func (s *MongoLoadTestStore) Update(test *domain.LoadTest) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	filter := bson.M{"id": test.ID}
	update := bson.M{"$set": test}
	
	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update load test: %w", err)
	}
	
	if result.MatchedCount == 0 {
		return fmt.Errorf("load test not found: %s", test.ID)
	}
	
	return nil
}

// List retrieves load tests based on filter criteria
func (s *MongoLoadTestStore) List(filter *LoadTestFilter) ([]*domain.LoadTest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	query := bson.M{}
	
	if filter != nil {
		if filter.AccountID != nil {
			query["accountId"] = *filter.AccountID
		}
		if filter.OrgID != nil {
			query["orgId"] = *filter.OrgID
		}
		if filter.ProjectID != nil {
			query["projectId"] = *filter.ProjectID
		}
		if filter.EnvID != nil {
			query["envId"] = *filter.EnvID
		}
		if len(filter.Tags) > 0 {
			query["tags"] = bson.M{"$in": filter.Tags}
		}
	}
	
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if filter != nil && filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	
	cursor, err := s.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list load tests: %w", err)
	}
	defer cursor.Close(ctx)
	
	var tests []*domain.LoadTest
	if err := cursor.All(ctx, &tests); err != nil {
		return nil, fmt.Errorf("failed to decode load tests: %w", err)
	}
	
	return tests, nil
}

// Delete deletes a load test by ID
func (s *MongoLoadTestStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := s.collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete load test: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("load test not found: %s", id)
	}
	
	return nil
}

// LoadTestRun methods

// createIndexes creates optimized indexes for load test runs
func (s *MongoLoadTestRunStore) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("id_unique_idx"),
		},
		{
			Keys:    bson.D{{Key: "loadTestId", Value: 1}},
			Options: options.Index().SetName("loadtest_idx"),
		},
		{
			Keys: bson.D{
				{Key: "loadTestId", Value: 1},
				{Key: "createdAt", Value: -1},
			},
			Options: options.Index().SetName("loadtest_created_idx"),
		},
		{
			Keys: bson.D{
				{Key: "accountId", Value: 1},
				{Key: "orgId", Value: 1},
				{Key: "projectId", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("account_org_project_status_idx"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}},
			Options: options.Index().SetName("status_idx"),
		},
		{
			Keys:    bson.D{{Key: "createdAt", Value: -1}},
			Options: options.Index().SetName("created_at_idx"),
		},
	}
	
	_, err := s.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create load test run indexes: %w", err)
	}
	
	return nil
}

// Create stores a new load test run
func (s *MongoLoadTestRunStore) Create(run *domain.LoadTestRun) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := s.collection.InsertOne(ctx, run)
	if err != nil {
		return fmt.Errorf("failed to create load test run: %w", err)
	}
	return nil
}

// Get retrieves a load test run by ID
func (s *MongoLoadTestRunStore) Get(id string) (*domain.LoadTestRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var run domain.LoadTestRun
	err := s.collection.FindOne(ctx, bson.M{"id": id}).Decode(&run)
	if err == mongo.ErrNoDocuments {
		return nil, fmt.Errorf("load test run not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get load test run: %w", err)
	}
	
	return &run, nil
}

// Update updates an existing load test run
func (s *MongoLoadTestRunStore) Update(run *domain.LoadTestRun) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	log.Printf("[MongoStore] Updating test run %s with status: %s", run.ID, run.Status)
	
	filter := bson.M{"id": run.ID}
	update := bson.M{"$set": run}
	
	result, err := s.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("[MongoStore] Update failed for run %s: %v", run.ID, err)
		return fmt.Errorf("failed to update load test run: %w", err)
	}
	
	log.Printf("[MongoStore] Update result for run %s: MatchedCount=%d, ModifiedCount=%d", 
		run.ID, result.MatchedCount, result.ModifiedCount)
	
	if result.MatchedCount == 0 {
		log.Printf("[MongoStore] No document matched for run %s", run.ID)
		return fmt.Errorf("load test run not found: %s", run.ID)
	}
	
	return nil
}

// List retrieves load test runs based on filter criteria
func (s *MongoLoadTestRunStore) List(filter *LoadTestRunFilter) ([]*domain.LoadTestRun, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	query := bson.M{}
	
	if filter != nil {
		if filter.LoadTestID != nil {
			query["loadTestId"] = *filter.LoadTestID
		}
		if filter.AccountID != nil {
			query["accountId"] = *filter.AccountID
		}
		if filter.OrgID != nil {
			query["orgId"] = *filter.OrgID
		}
		if filter.ProjectID != nil {
			query["projectId"] = *filter.ProjectID
		}
		if filter.EnvID != nil {
			query["envId"] = *filter.EnvID
		}
		if filter.Status != nil {
			query["status"] = *filter.Status
		}
	}
	
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	if filter != nil && filter.Limit > 0 {
		opts.SetLimit(int64(filter.Limit))
	}
	
	cursor, err := s.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list load test runs: %w", err)
	}
	defer cursor.Close(ctx)
	
	var runs []*domain.LoadTestRun
	if err := cursor.All(ctx, &runs); err != nil {
		return nil, fmt.Errorf("failed to decode load test runs: %w", err)
	}
	
	return runs, nil
}

// Delete deletes a load test run by ID
func (s *MongoLoadTestRunStore) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	result, err := s.collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		return fmt.Errorf("failed to delete load test run: %w", err)
	}
	
	if result.DeletedCount == 0 {
		return fmt.Errorf("load test run not found: %s", id)
	}
	
	return nil
}
