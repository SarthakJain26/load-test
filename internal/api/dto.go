package api

import "Load-manager-cli/internal/domain"

// CreateTestRequest represents the request body for creating a test run
type CreateTestRequest struct {
	TenantID        string         `json:"tenantId" binding:"required"`
	EnvID           string         `json:"envId" binding:"required"`
	ScenarioID      string         `json:"scenarioId" binding:"required"`
	TargetUsers     int            `json:"targetUsers" binding:"required,min=1"`
	SpawnRate       float64        `json:"spawnRate" binding:"required,min=0.1"`
	DurationSeconds *int           `json:"durationSeconds,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// TestRunResponse represents the response body for a test run
type TestRunResponse struct {
	ID              string                  `json:"id"`
	TenantID        string                  `json:"tenantId"`
	EnvID           string                  `json:"envId"`
	LocustClusterID string                  `json:"locustClusterId"`
	ScenarioID      string                  `json:"scenarioId"`
	TargetUsers     int                     `json:"targetUsers"`
	SpawnRate       float64                 `json:"spawnRate"`
	DurationSeconds *int                    `json:"durationSeconds,omitempty"`
	Status          string                  `json:"status"`
	StartedAt       *string                 `json:"startedAt,omitempty"`
	FinishedAt      *string                 `json:"finishedAt,omitempty"`
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
}

// LocustCallbackMetricsRequest represents the callback payload for periodic metrics
type LocustCallbackMetricsRequest struct {
	RunID   string                  `json:"runId" binding:"required"`
	Metrics *MetricSnapshotResponse `json:"metrics" binding:"required"`
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

func toTestRunResponse(run *domain.TestRun) *TestRunResponse {
	resp := &TestRunResponse{
		ID:              run.ID,
		TenantID:        run.TenantID,
		EnvID:           run.EnvID,
		LocustClusterID: run.LocustClusterID,
		ScenarioID:      run.ScenarioID,
		TargetUsers:     run.TargetUsers,
		SpawnRate:       run.SpawnRate,
		DurationSeconds: run.DurationSeconds,
		Status:          string(run.Status),
		Metadata:        run.Metadata,
	}
	
	if run.StartedAt != nil {
		startedAt := run.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &startedAt
	}
	
	if run.FinishedAt != nil {
		finishedAt := run.FinishedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.FinishedAt = &finishedAt
	}
	
	if run.LastMetrics != nil {
		resp.LastMetrics = toMetricSnapshotResponse(run.LastMetrics)
	}
	
	return resp
}

func toMetricSnapshotResponse(metrics *domain.MetricSnapshot) *MetricSnapshotResponse {
	resp := &MetricSnapshotResponse{
		Timestamp:         metrics.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
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
