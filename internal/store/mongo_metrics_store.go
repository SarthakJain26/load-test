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
	metricsTimeseriesCollection = "metrics_timeseries"
)

// MetricsDocument represents a time-series metrics document
type MetricsDocument struct {
	Timestamp       int64                  `bson:"timestamp"` // Unix milliseconds
	LoadTestRunID   string                 `bson:"loadTestRunId"`
	AccountID       string                 `bson:"accountId"`
	OrgID           string                 `bson:"orgId"`
	ProjectID       string                 `bson:"projectId"`
	EnvID           string                 `bson:"envId,omitempty"`
	TotalRPS        float64                `bson:"totalRps"`
	TotalRequests   int64                  `bson:"totalRequests"`
	TotalFailures   int64                  `bson:"totalFailures"`
	ErrorRate       float64                `bson:"errorRate"`
	CurrentUsers    int                    `bson:"currentUsers"`
	P50ResponseMs   float64                `bson:"p50ResponseMs"`
	P95ResponseMs   float64                `bson:"p95ResponseMs"`
	P99ResponseMs   float64                `bson:"p99ResponseMs"`
	MinResponseMs   float64                `bson:"minResponseMs"`
	MaxResponseMs   float64                `bson:"maxResponseMs"`
	AvgResponseMs   float64                `bson:"avgResponseMs"`
	RequestStats    []RequestStatDocument  `bson:"requestStats"`
	Metadata        map[string]interface{} `bson:"metadata,omitempty"`
}

// RequestStatDocument represents per-endpoint stats
type RequestStatDocument struct {
	Method            string  `bson:"method"`
	Name              string  `bson:"name"`
	NumRequests       int64   `bson:"numRequests"`
	NumFailures       int64   `bson:"numFailures"`
	AvgResponseTimeMs float64 `bson:"avgResponseTimeMs"`
	MinResponseTimeMs float64 `bson:"minResponseTimeMs"`
	MaxResponseTimeMs float64 `bson:"maxResponseTimeMs"`
	P50ResponseMs     float64 `bson:"p50ResponseMs"`
	P95ResponseMs     float64 `bson:"p95ResponseMs"`
	RequestsPerSec    float64 `bson:"requestsPerSec"`
}

// MongoMetricsStore handles time-series metrics storage
type MongoMetricsStore struct {
	collection *mongo.Collection
}

// NewMongoMetricsStore creates a new time-series metrics store
func NewMongoMetricsStore(db *mongo.Database) (*MongoMetricsStore, error) {
	collection := db.Collection(metricsTimeseriesCollection)

	store := &MongoMetricsStore{
		collection: collection,
	}

	if err := store.ensureTimeseriesCollection(db); err != nil {
		return nil, err
	}

	if err := store.createIndexes(); err != nil {
		return nil, err
	}

	return store, nil
}

// ensureTimeseriesCollection creates the time-series collection if needed
func (s *MongoMetricsStore) ensureTimeseriesCollection(db *mongo.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collections, err := db.ListCollectionNames(ctx, bson.M{"name": metricsTimeseriesCollection})
	if err != nil {
		return fmt.Errorf("failed to list collections: %w", err)
	}

	if len(collections) > 0 {
		return nil
	}

	opts := options.CreateCollection().
		SetTimeSeriesOptions(options.TimeSeries().
			SetTimeField("timestamp").
			SetMetaField("testRunId").
			SetGranularity("seconds"))

	if err := db.CreateCollection(ctx, metricsTimeseriesCollection, opts); err != nil {
		return fmt.Errorf("failed to create time-series collection: %w", err)
	}

	return nil
}

// createIndexes creates optimized indexes
func (s *MongoMetricsStore) createIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	indexes := []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "loadTestRunId", Value: 1},
				{Key: "timestamp", Value: 1},
			},
			Options: options.Index().SetName("loadtestrun_timestamp_idx"),
		},
		{
			Keys: bson.D{
				{Key: "accountId", Value: 1},
				{Key: "orgId", Value: 1},
				{Key: "projectId", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().SetName("account_org_project_timestamp_idx"),
		},
		{
			Keys: bson.D{
				{Key: "loadTestRunId", Value: 1},
			},
			Options: options.Index().SetName("loadtestrun_idx"),
		},
	}

	_, err := s.collection.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

// StoreMetric stores a metric snapshot
func (s *MongoMetricsStore) StoreMetric(ctx context.Context, loadTestRunID, accountID, orgID, projectID, envID string, metric *domain.MetricSnapshot) error {
	doc := MetricsDocument{
		Timestamp:     metric.Timestamp, // Already int64 Unix milliseconds
		LoadTestRunID: loadTestRunID,
		AccountID:     accountID,
		OrgID:         orgID,
		ProjectID:     projectID,
		EnvID:         envID,
		TotalRPS:      metric.TotalRPS,
		TotalRequests: metric.TotalRequests,
		TotalFailures: metric.TotalFailures,
		ErrorRate:     metric.ErrorRate,
		CurrentUsers:  metric.CurrentUsers,
		P50ResponseMs: metric.P50ResponseMs,
		P95ResponseMs: metric.P95ResponseMs,
		P99ResponseMs: metric.P99ResponseMs,
		MinResponseMs: metric.MinResponseMs,
		MaxResponseMs: metric.MaxResponseMs,
		AvgResponseMs: metric.AvgResponseMs,
		RequestStats:  make([]RequestStatDocument, 0, len(metric.RequestStats)),
	}

	for _, stat := range metric.RequestStats {
		if stat != nil {
			doc.RequestStats = append(doc.RequestStats, RequestStatDocument{
				Method:            stat.Method,
				Name:              stat.Name,
				NumRequests:       stat.NumRequests,
				NumFailures:       stat.NumFailures,
				AvgResponseTimeMs: stat.AvgResponseTimeMs,
				MinResponseTimeMs: stat.MinResponseTimeMs,
				MaxResponseTimeMs: stat.MaxResponseTimeMs,
				P50ResponseMs:     stat.P50ResponseMs,
				P95ResponseMs:     stat.P95ResponseMs,
				RequestsPerSec:    stat.RequestsPerSec,
			})
		}
	}

	_, err := s.collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("failed to insert metric: %w", err)
	}

	return nil
}

// GetMetricsTimeseries retrieves time-series data for charts
func (s *MongoMetricsStore) GetMetricsTimeseries(ctx context.Context, loadTestRunID string, fromTime, toTime int64) ([]MetricsDocument, error) {
	filter := bson.M{
		"loadTestRunId": loadTestRunID,
	}

	if fromTime > 0 || toTime > 0 {
		timeFilter := bson.M{}
		if fromTime > 0 {
			timeFilter["$gte"] = fromTime
		}
		if toTime > 0 {
			timeFilter["$lte"] = toTime
		}
		filter["timestamp"] = timeFilter
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := s.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []MetricsDocument
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode metrics: %w", err)
	}

	return results, nil
}

// GetAggregatedMetrics retrieves aggregated metrics for a test run
func (s *MongoMetricsStore) GetAggregatedMetrics(ctx context.Context, loadTestRunID string) (*AggregatedMetrics, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"loadTestRunId": loadTestRunID}}},
		{{Key: "$group", Value: bson.M{
			"_id":           nil,
			"avgRPS":        bson.M{"$avg": "$totalRps"},
			"maxRPS":        bson.M{"$max": "$totalRps"},
			"minRPS":        bson.M{"$min": "$totalRps"},
			"avgP50":        bson.M{"$avg": "$p50ResponseMs"},
			"avgP95":        bson.M{"$avg": "$p95ResponseMs"},
			"avgP99":        bson.M{"$avg": "$p99ResponseMs"},
			"maxP95":        bson.M{"$max": "$p95ResponseMs"},
			"totalRequests": bson.M{"$sum": "$totalRequests"},
			"totalFailures": bson.M{"$sum": "$totalFailures"},
			"dataPoints":    bson.M{"$sum": 1},
		}}},
	}

	cursor, err := s.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []AggregatedMetrics
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode aggregated metrics: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no metrics found for test run")
	}

	return &results[0], nil
}

// AggregatedMetrics holds aggregated statistics
type AggregatedMetrics struct {
	AvgRPS        float64 `bson:"avgRPS"`
	MaxRPS        float64 `bson:"maxRPS"`
	MinRPS        float64 `bson:"minRPS"`
	AvgP50        float64 `bson:"avgP50"`
	AvgP95        float64 `bson:"avgP95"`
	AvgP99        float64 `bson:"avgP99"`
	MaxP95        float64 `bson:"maxP95"`
	TotalRequests int64   `bson:"totalRequests"`
	TotalFailures int64   `bson:"totalFailures"`
	DataPoints    int     `bson:"dataPoints"`
}
