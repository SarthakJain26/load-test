package domain

import "time"

// TestRunStatus represents the current status of a load test run
type TestRunStatus string

const (
	TestRunStatusPending  TestRunStatus = "Pending"
	TestRunStatusRunning  TestRunStatus = "Running"
	TestRunStatusStopping TestRunStatus = "Stopping"
	TestRunStatusFinished TestRunStatus = "Finished"
	TestRunStatusFailed   TestRunStatus = "Failed"
)

// Tenant represents a tenant in the system (for multi-tenancy support)
type Tenant struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Environment represents an environment within a tenant (e.g., dev, staging, prod)
type Environment struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // e.g., "dev", "staging", "production"
	TenantID string `json:"tenantId"`
}

// LocustCluster represents a Locust master cluster configuration
type LocustCluster struct {
	ID        string `json:"id"`
	BaseURL   string `json:"baseUrl"`   // e.g., "http://locust-master:8089"
	TenantID  string `json:"tenantId"`
	EnvID     string `json:"envId"`
	AuthToken string `json:"authToken"` // Optional API key/token for Locust master
}

// TestRun represents a load test execution
type TestRun struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenantId"`
	EnvID           string         `json:"envId"`
	LocustClusterID string         `json:"locustClusterId"`
	ScenarioID      string         `json:"scenarioId"` // Maps to locustfile/scenario/tag
	TargetUsers     int            `json:"targetUsers"`
	SpawnRate       float64        `json:"spawnRate"`
	DurationSeconds *int           `json:"durationSeconds,omitempty"` // Optional duration
	Status          TestRunStatus  `json:"status"`
	StartedAt       *time.Time     `json:"startedAt,omitempty"`
	FinishedAt      *time.Time     `json:"finishedAt,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"` // Additional metadata (notes, tags, etc.)
	LastMetrics     *MetricSnapshot `json:"lastMetrics,omitempty"`
}

// MetricSnapshot represents aggregated metrics from Locust at a point in time
type MetricSnapshot struct {
	Timestamp         time.Time          `json:"timestamp"`
	TotalRPS          float64            `json:"totalRps"`
	TotalRequests     int64              `json:"totalRequests"`
	TotalFailures     int64              `json:"totalFailures"`
	ErrorRate         float64            `json:"errorRate"` // Percentage
	AverageResponseMs float64            `json:"avgResponseMs"`
	MinResponseMs     float64            `json:"minResponseMs"`
	MaxResponseMs     float64            `json:"maxResponseMs"`
	AvgResponseMs     float64            `json:"avgResponseMs"` // Alias for AverageResponseMs
	P50ResponseMs     float64            `json:"p50ResponseMs"`
	P95ResponseMs     float64            `json:"p95ResponseMs"`
	P99ResponseMs     float64            `json:"p99ResponseMs"`
	CurrentUsers      int                `json:"currentUsers"`
	RequestStats      map[string]*ReqStat `json:"requestStats,omitempty"` // Per-endpoint stats
}

// ReqStat represents statistics for a specific request/endpoint
type ReqStat struct {
	Method             string  `json:"method"`
	Name               string  `json:"name"`
	NumRequests        int64   `json:"numRequests"`
	NumFailures        int64   `json:"numFailures"`
	AvgResponseTime    float64 `json:"avgResponseTime"`
	AvgResponseTimeMs  float64 `json:"avgResponseTimeMs"` // In milliseconds
	MinResponseTime    float64 `json:"minResponseTime"`
	MinResponseTimeMs  float64 `json:"minResponseTimeMs"` // In milliseconds
	MaxResponseTime    float64 `json:"maxResponseTime"`
	MaxResponseTimeMs  float64 `json:"maxResponseTimeMs"` // In milliseconds
	MedianResponseTime float64 `json:"medianResponseTime"`
	P50ResponseMs      float64 `json:"p50ResponseMs"` // 50th percentile
	P95ResponseMs      float64 `json:"p95ResponseMs"` // 95th percentile
	RequestsPerSec     float64 `json:"requestsPerSec"`
}
