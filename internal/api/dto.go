package api

import (
	"Load-manager-cli/internal/domain"
	"time"
)

// LoadTest DTOs

// CreateLoadTestRequest represents the request body for creating a load test
type CreateLoadTestRequest struct {
	Name               string         `json:"name" binding:"required"`
	Description        string         `json:"description,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	AccountID          string         `json:"accountId" binding:"required"`
	OrgID              string         `json:"orgId" binding:"required"`
	ProjectID          string         `json:"projectId" binding:"required"`
	EnvID              string         `json:"envId,omitempty"`
	LocustClusterID    string         `json:"locustClusterId" binding:"required"`
	TargetURL          string         `json:"targetUrl" binding:"required"`
	ScriptContent      string         `json:"scriptContent" binding:"required"` // Base64 encoded Python script
	ScenarioID         string         `json:"scenarioId,omitempty"`
	DefaultUsers       int            `json:"defaultUsers,omitempty"`
	DefaultSpawnRate   float64        `json:"defaultSpawnRate,omitempty"`
	DefaultDurationSec *int           `json:"defaultDurationSec,omitempty"`
	MaxDurationSec     *int           `json:"maxDurationSec,omitempty"`
	CreatedBy          string         `json:"createdBy" binding:"required"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

// UpdateLoadTestRequest represents the request body for updating a load test
type UpdateLoadTestRequest struct {
	Name               string         `json:"name,omitempty"`
	Description        string         `json:"description,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	TargetURL          string         `json:"targetUrl,omitempty"`
	ScenarioID         string         `json:"scenarioId,omitempty"`
	DefaultUsers       int            `json:"defaultUsers,omitempty"`
	DefaultSpawnRate   float64        `json:"defaultSpawnRate,omitempty"`
	DefaultDurationSec *int           `json:"defaultDurationSec,omitempty"`
	MaxDurationSec     *int           `json:"maxDurationSec,omitempty"`
	UpdatedBy          string         `json:"updatedBy" binding:"required"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

// UpdateScriptRequest represents the request body for updating a load test script (creates new revision)
type UpdateScriptRequest struct {
	ScriptContent string `json:"scriptContent" binding:"required"` // Base64 encoded Python script
	Description   string `json:"description,omitempty"`            // Optional change description
	UpdatedBy     string `json:"updatedBy" binding:"required"`
}

// LoadTestResponse represents the response body for a load test
type LoadTestResponse struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	AccountID          string         `json:"accountId"`
	OrgID              string         `json:"orgId"`
	ProjectID          string         `json:"projectId"`
	EnvID              string         `json:"envId,omitempty"`
	LocustClusterID    string         `json:"locustClusterId"`
	TargetURL          string         `json:"targetUrl"`
	LatestRevisionID   string         `json:"latestRevisionId,omitempty"`
	ScenarioID         string         `json:"scenarioId,omitempty"`
	DefaultUsers       int            `json:"defaultUsers,omitempty"`
	DefaultSpawnRate   float64        `json:"defaultSpawnRate,omitempty"`
	DefaultDurationSec *int           `json:"defaultDurationSec,omitempty"`
	MaxDurationSec     *int           `json:"maxDurationSec,omitempty"`
	CreatedAt          string         `json:"createdAt"`
	CreatedBy          string         `json:"createdBy"`
	UpdatedAt          string         `json:"updatedAt"`
	UpdatedBy          string         `json:"updatedBy"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

// ScriptRevisionResponse represents the response body for a script revision
type ScriptRevisionResponse struct {
	ID             string `json:"id"`
	LoadTestID     string `json:"loadTestId"`
	RevisionNumber int    `json:"revisionNumber"`
	ScriptContent  string `json:"scriptContent"` // Base64 encoded Python script
	Description    string `json:"description,omitempty"`
	CreatedAt      string `json:"createdAt"`
	CreatedBy      string `json:"createdBy"`
}

// LoadTestRun DTOs

// CreateLoadTestRunRequest represents the request body for creating/starting a load test run
type CreateLoadTestRunRequest struct {
	LoadTestID      string         `json:"loadTestId" binding:"required"`
	Name            string         `json:"name,omitempty"`
	TargetUsers     *int           `json:"targetUsers,omitempty"`     // Override from LoadTest
	SpawnRate       *float64       `json:"spawnRate,omitempty"`       // Override from LoadTest
	DurationSeconds *int           `json:"durationSeconds,omitempty"` // Override from LoadTest
	CreatedBy       string         `json:"createdBy" binding:"required"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// LoadTestRunResponse represents the response body for a load test run
type LoadTestRunResponse struct {
	ID              string                  `json:"id"`
	LoadTestID      string                  `json:"loadTestId"`
	Name            string                  `json:"name,omitempty"`
	AccountID       string                  `json:"accountId"`
	OrgID           string                  `json:"orgId"`
	ProjectID       string                  `json:"projectId"`
	EnvID           string                  `json:"envId,omitempty"`
	TargetUsers     int                     `json:"targetUsers"`
	SpawnRate       float64                 `json:"spawnRate"`
	DurationSeconds *int                    `json:"durationSeconds,omitempty"`
	Status          string                  `json:"status"`
	StartedAt       *string                 `json:"startedAt,omitempty"`
	FinishedAt      *string                 `json:"finishedAt,omitempty"`
	CreatedAt       string                  `json:"createdAt"`
	CreatedBy       string                  `json:"createdBy"`
	UpdatedAt       string                  `json:"updatedAt"`
	UpdatedBy       string                  `json:"updatedBy"`
	Metadata        map[string]any          `json:"metadata,omitempty"`
	LastMetrics     *MetricSnapshotResponse `json:"lastMetrics,omitempty"`
}

// MetricSnapshotResponse represents metrics data in API response
type MetricSnapshotResponse struct {
	Timestamp         string                  `json:"timestamp"`
	TotalRPS          float64                 `json:"totalRps"`
	TotalRequests     int64                   `json:"totalRequests"`
	TotalFailures     int64                   `json:"totalFailures"`
	ErrorRate         float64                 `json:"errorRate"`
	AverageResponseMs float64                 `json:"avgResponseMs"`
	P50ResponseMs     float64                 `json:"p50ResponseMs"`
	P95ResponseMs     float64                 `json:"p95ResponseMs"`
	P99ResponseMs     float64                 `json:"p99ResponseMs"`
	CurrentUsers      int                     `json:"currentUsers"`
	RequestStats      map[string]*ReqStatResponse `json:"requestStats,omitempty"`
}

// ReqStatResponse represents per-request statistics in API response
type ReqStatResponse struct {
	Method             string  `json:"method"`
	Name               string  `json:"name"`
	NumRequests        int64   `json:"numRequests"`
	NumFailures        int64   `json:"numFailures"`
	AvgResponseTime    float64 `json:"avgResponseTime"`
	MinResponseTime    float64 `json:"minResponseTime"`
	MaxResponseTime    float64 `json:"maxResponseTime"`
	MedianResponseTime float64 `json:"medianResponseTime"`
	RequestsPerSec     float64 `json:"requestsPerSec"`
}

// LocustCallbackTestStartRequest represents the callback payload when test starts
type LocustCallbackTestStartRequest struct {
	RunID    string `json:"runId" binding:"required"`
	TenantID string `json:"tenantId"`
	EnvID    string `json:"envId"`
}

// LocustCallbackTestStopRequest represents the callback payload when test stops
type LocustCallbackTestStopRequest struct {
	RunID        string                  `json:"runId" binding:"required"`
	TenantID     string                  `json:"tenantId"`
	EnvID        string                  `json:"envId"`
	FinalMetrics *MetricSnapshotResponse `json:"finalMetrics,omitempty"`
	AutoStopped  bool                    `json:"autoStopped"` // True if stopped by duration, false if manual
}

// LocustCallbackMetricsRequest represents the callback payload for periodic metrics
type LocustCallbackMetricsRequest struct {
	RunID   string                  `json:"runId" binding:"required"`
	Metrics *MetricSnapshotResponse `json:"metrics" binding:"required"`
}

// RegisterExternalTestRequest is used when a test is started directly from Locust UI
type RegisterExternalTestRequest struct {
	AccountID   string  `json:"accountId" binding:"required"`
	OrgID       string  `json:"orgId" binding:"required"`
	ProjectID   string  `json:"projectId" binding:"required"`
	EnvID       string  `json:"envId,omitempty"`
	ScenarioID  string  `json:"scenarioId"`
	TargetUsers int     `json:"targetUsers"`
	SpawnRate   float64 `json:"spawnRate"`
}

// RegisterExternalTestResponse returns the assigned run ID
type RegisterExternalTestResponse struct {
	RunID   string `json:"runId"`
	Message string `json:"message"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a generic success response
type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

// Helper functions to convert between domain models and DTOs

// LoadTest conversions

func toLoadTestResponse(test *domain.LoadTest) *LoadTestResponse {
	return &LoadTestResponse{
		ID:                 test.ID,
		Name:               test.Name,
		Description:        test.Description,
		Tags:               test.Tags,
		AccountID:          test.AccountID,
		OrgID:              test.OrgID,
		ProjectID:          test.ProjectID,
		EnvID:              test.EnvID,
		LocustClusterID:    test.LocustClusterID,
		TargetURL:          test.TargetURL,
		LatestRevisionID:   test.LatestRevisionID,
		ScenarioID:         test.ScenarioID,
		DefaultUsers:       test.DefaultUsers,
		DefaultSpawnRate:   test.DefaultSpawnRate,
		DefaultDurationSec: test.DefaultDurationSec,
		MaxDurationSec:     test.MaxDurationSec,
		CreatedAt:          time.UnixMilli(test.CreatedAt).Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy:          test.CreatedBy,
		UpdatedAt:          time.UnixMilli(test.UpdatedAt).Format("2006-01-02T15:04:05Z07:00"),
		UpdatedBy:          test.UpdatedBy,
		Metadata:           test.Metadata,
	}
}

// ScriptRevision conversions

func toScriptRevisionResponse(revision *domain.ScriptRevision) *ScriptRevisionResponse {
	return &ScriptRevisionResponse{
		ID:             revision.ID,
		LoadTestID:     revision.LoadTestID,
		RevisionNumber: revision.RevisionNumber,
		ScriptContent:  revision.ScriptContent,
		Description:    revision.Description,
		CreatedAt:      time.UnixMilli(revision.CreatedAt).Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy:      revision.CreatedBy,
	}
}

// LoadTestRun conversions

func toLoadTestRunResponse(run *domain.LoadTestRun) *LoadTestRunResponse {
	resp := &LoadTestRunResponse{
		ID:              run.ID,
		LoadTestID:      run.LoadTestID,
		Name:            run.Name,
		AccountID:       run.AccountID,
		OrgID:           run.OrgID,
		ProjectID:       run.ProjectID,
		EnvID:           run.EnvID,
		TargetUsers:     run.TargetUsers,
		SpawnRate:       run.SpawnRate,
		DurationSeconds: run.DurationSeconds,
		Status:          string(run.Status),
		CreatedAt:       time.UnixMilli(run.CreatedAt).Format("2006-01-02T15:04:05Z07:00"),
		CreatedBy:       run.CreatedBy,
		UpdatedAt:       time.UnixMilli(run.UpdatedAt).Format("2006-01-02T15:04:05Z07:00"),
		UpdatedBy:       run.UpdatedBy,
		Metadata:        run.Metadata,
	}
	
	if run.StartedAt > 0 {
		startedAt := time.UnixMilli(run.StartedAt).Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &startedAt
	}
	
	if run.FinishedAt > 0 {
		finishedAt := time.UnixMilli(run.FinishedAt).Format("2006-01-02T15:04:05Z07:00")
		resp.FinishedAt = &finishedAt
	}
	
	if run.LastMetrics != nil {
		resp.LastMetrics = toMetricSnapshotResponse(run.LastMetrics)
	}
	
	return resp
}

func toMetricSnapshotResponse(metrics *domain.MetricSnapshot) *MetricSnapshotResponse {
	resp := &MetricSnapshotResponse{
		Timestamp:         time.UnixMilli(metrics.Timestamp).Format("2006-01-02T15:04:05Z07:00"),
		TotalRPS:          metrics.TotalRPS,
		TotalRequests:     metrics.TotalRequests,
		TotalFailures:     metrics.TotalFailures,
		ErrorRate:         metrics.ErrorRate,
		AverageResponseMs: metrics.AverageResponseMs,
		P50ResponseMs:     metrics.P50ResponseMs,
		P95ResponseMs:     metrics.P95ResponseMs,
		P99ResponseMs:     metrics.P99ResponseMs,
		CurrentUsers:      metrics.CurrentUsers,
	}
	
	if metrics.RequestStats != nil {
		resp.RequestStats = make(map[string]*ReqStatResponse)
		for k, v := range metrics.RequestStats {
			if v != nil {
				resp.RequestStats[k] = &ReqStatResponse{
					Method:             v.Method,
					Name:               v.Name,
					NumRequests:        v.NumRequests,
					NumFailures:        v.NumFailures,
					AvgResponseTime:    v.AvgResponseTime,
					MinResponseTime:    v.MinResponseTime,
					MaxResponseTime:    v.MaxResponseTime,
					MedianResponseTime: v.MedianResponseTime,
					RequestsPerSec:     v.RequestsPerSec,
				}
			}
		}
	}
	
	return resp
}

func toDomainMetricSnapshot(resp *MetricSnapshotResponse) *domain.MetricSnapshot {
	if resp == nil {
		return nil
	}
	
	metrics := &domain.MetricSnapshot{
		TotalRPS:          resp.TotalRPS,
		TotalRequests:     resp.TotalRequests,
		TotalFailures:     resp.TotalFailures,
		ErrorRate:         resp.ErrorRate,
		AverageResponseMs: resp.AverageResponseMs,
		P50ResponseMs:     resp.P50ResponseMs,
		P95ResponseMs:     resp.P95ResponseMs,
		P99ResponseMs:     resp.P99ResponseMs,
		CurrentUsers:      resp.CurrentUsers,
	}
	
	if resp.RequestStats != nil {
		metrics.RequestStats = make(map[string]*domain.ReqStat)
		for k, v := range resp.RequestStats {
			if v != nil {
				metrics.RequestStats[k] = &domain.ReqStat{
					Method:             v.Method,
					Name:               v.Name,
					NumRequests:        v.NumRequests,
					NumFailures:        v.NumFailures,
					AvgResponseTime:    v.AvgResponseTime,
					MinResponseTime:    v.MinResponseTime,
					MaxResponseTime:    v.MaxResponseTime,
					MedianResponseTime: v.MedianResponseTime,
					RequestsPerSec:     v.RequestsPerSec,
				}
			}
		}
	}
	
	return metrics
}
