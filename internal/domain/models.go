package domain

// LoadTestRunStatus represents the current status of a load test run
type LoadTestRunStatus string

const (
	LoadTestRunStatusPending  LoadTestRunStatus = "Pending"
	LoadTestRunStatusRunning  LoadTestRunStatus = "Running"
	LoadTestRunStatusStopping LoadTestRunStatus = "Stopping"
	LoadTestRunStatusStopped  LoadTestRunStatus = "Stopped"  // Manual stop
	LoadTestRunStatusFinished LoadTestRunStatus = "Finished" // Auto-completed
	LoadTestRunStatusFailed   LoadTestRunStatus = "Failed"
)

// LocustCluster represents a Locust master cluster configuration
type LocustCluster struct {
	ID        string `json:"id"`
	BaseURL   string `json:"baseUrl"`   // e.g., "http://locust-master:8089"
	AccountID string `json:"accountId"`
	OrgID     string `json:"orgId"`
	ProjectID string `json:"projectId"`
	EnvID     string `json:"envId,omitempty"` // Optional environment
	AuthToken string `json:"authToken"`       // Optional API key/token for Locust master
}

// ScriptRevision represents a version of a Locust test script
type ScriptRevision struct {
	ID             string `json:"id" bson:"id"`                                     // Unique revision ID
	LoadTestID     string `json:"loadTestId" bson:"loadTestId"`                     // Reference to the LoadTest
	RevisionNumber int    `json:"revisionNumber" bson:"revisionNumber"`             // Sequential revision number (1, 2, 3, ...)
	ScriptContent  string `json:"scriptContent" bson:"scriptContent"`               // Base64 encoded Python script
	Description    string `json:"description,omitempty" bson:"description,omitempty"` // Optional change description
	CreatedAt      int64  `json:"createdAt" bson:"createdAt"`                       // Unix milliseconds
	CreatedBy      string `json:"createdBy" bson:"createdBy"`
}

// RecentRun represents a summary of a recent LoadTestRun execution
type RecentRun struct {
	ID              string            `json:"id"`
	Name            string            `json:"name,omitempty"`
	Status          LoadTestRunStatus `json:"status"`
	TargetUsers     int               `json:"targetUsers"`
	SpawnRate       float64           `json:"spawnRate"`
	DurationSeconds *int              `json:"durationSeconds,omitempty"`
	StartedAt       int64             `json:"startedAt,omitempty"`  // Unix milliseconds
	FinishedAt      int64             `json:"finishedAt,omitempty"` // Unix milliseconds
	CreatedAt       int64             `json:"createdAt"`            // Unix milliseconds
	CreatedBy       string            `json:"createdBy"`
}

// LoadTest represents a load test definition/template
type LoadTest struct {
	ID              string         `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	AccountID       string         `json:"accountId"`
	OrgID           string         `json:"orgId"`
	ProjectID       string         `json:"projectId"`
	EnvID           string         `json:"envId,omitempty"`     // Optional environment
	LocustClusterID string         `json:"locustClusterId"`
	TargetURL       string         `json:"targetUrl"`
	LatestRevisionID string        `json:"latestRevisionId,omitempty"` // Reference to the latest script revision
	ScenarioID      string         `json:"scenarioId,omitempty"` // Optional scenario/tag within locustfile
	// Default runtime parameters
	DefaultUsers         int     `json:"defaultUsers,omitempty"`
	DefaultSpawnRate     float64 `json:"defaultSpawnRate,omitempty"`
	DefaultDurationSec   *int    `json:"defaultDurationSec,omitempty"`
	MaxDurationSec       *int    `json:"maxDurationSec,omitempty"` // Maximum allowed duration
	// Recent runs (up to 10 most recent)
	RecentRuns []RecentRun `json:"recentRuns,omitempty"`
	// Audit fields (Unix milliseconds)
	CreatedAt  int64  `json:"createdAt"`
	CreatedBy  string `json:"createdBy"`
	UpdatedAt  int64  `json:"updatedAt"`
	UpdatedBy  string `json:"updatedBy"`
	Metadata   map[string]any `json:"metadata,omitempty"` // Additional metadata
}

// LoadTestRun represents an actual execution of a load test
type LoadTestRun struct {
	ID               string `json:"id"`
	LoadTestID       string `json:"loadTestId"` // Reference to the LoadTest
	ScriptRevisionID string `json:"scriptRevisionId"` // Reference to the script revision used for this run
	Name             string `json:"name,omitempty"` // Optional run name
	AccountID        string `json:"accountId"`
	OrgID            string `json:"orgId"`
	ProjectID        string `json:"projectId"`
	EnvID            string `json:"envId,omitempty"` // Optional environment
	// Runtime parameters (can override LoadTest defaults)
	TargetUsers     int     `json:"targetUsers"`
	SpawnRate       float64 `json:"spawnRate"`
	DurationSeconds *int    `json:"durationSeconds,omitempty"`
	// Execution state
	Status       LoadTestRunStatus `json:"status"`
	StartedAt    int64             `json:"startedAt,omitempty"`  // Unix milliseconds
	FinishedAt   int64             `json:"finishedAt,omitempty"` // Unix milliseconds
	LastMetrics  *MetricSnapshot   `json:"lastMetrics,omitempty"`
	// Audit fields (Unix milliseconds)
	CreatedAt int64          `json:"createdAt"`
	CreatedBy string         `json:"createdBy"`
	UpdatedAt int64          `json:"updatedAt"`
	UpdatedBy string         `json:"updatedBy"`
	Metadata  map[string]any `json:"metadata,omitempty"` // Additional run metadata
}

// MetricSnapshot represents aggregated metrics from Locust at a point in time
type MetricSnapshot struct {
	Timestamp         int64              `json:"timestamp"` // Unix milliseconds
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
