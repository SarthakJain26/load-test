package api

import (
	"time"
)

// TimeseriesDataPoint represents a single point in time-series chart
type TimeseriesDataPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	TotalRPS      float64   `json:"totalRps"`
	CurrentUsers  int       `json:"currentUsers"`
	P50ResponseMs float64   `json:"p50ResponseMs"`
	P95ResponseMs float64   `json:"p95ResponseMs"`
	P99ResponseMs float64   `json:"p99ResponseMs"`
	ErrorRate     float64   `json:"errorRate"`
}

// TimeseriesChartResponse is for line charts (RPS, latency over time)
type TimeseriesChartResponse struct {
	TestRunID  string                `json:"testRunId"`
	DataPoints []TimeseriesDataPoint `json:"dataPoints"`
	Summary    AggregatedSummary     `json:"summary"`
}

// ScatterDataPoint represents a single request for scatter plot
type ScatterDataPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Endpoint       string    `json:"endpoint"`
	Method         string    `json:"method"`
	ResponseTimeMs float64   `json:"responseTimeMs"`
	Success        bool      `json:"success"`
}

// ScatterPlotResponse is for scatter plots (response time distribution)
type ScatterPlotResponse struct {
	TestRunID  string             `json:"testRunId"`
	DataPoints []ScatterDataPoint `json:"dataPoints"`
	Endpoints  []string           `json:"endpoints"`
}

// AggregatedSummary provides aggregated statistics
type AggregatedSummary struct {
	AvgRPS           float64 `json:"avgRps"`
	MaxRPS           float64 `json:"maxRps"`
	MinRPS           float64 `json:"minRps"`
	AvgP50Latency    float64 `json:"avgP50Latency"`
	AvgP95Latency    float64 `json:"avgP95Latency"`
	AvgP99Latency    float64 `json:"avgP99Latency"`
	MaxP95Latency    float64 `json:"maxP95Latency"`
	TotalRequests    int64   `json:"totalRequests"`
	TotalFailures    int64   `json:"totalFailures"`
	OverallErrorRate float64 `json:"overallErrorRate"`
	DataPoints       int     `json:"dataPoints"`
	Duration         string  `json:"duration"`
}

// EndpointStatsResponse provides per-endpoint statistics
type EndpointStatsResponse struct {
	Endpoint          string  `json:"endpoint"`
	Method            string  `json:"method"`
	TotalRequests     int64   `json:"totalRequests"`
	TotalFailures     int64   `json:"totalFailures"`
	ErrorRate         float64 `json:"errorRate"`
	AvgResponseTimeMs float64 `json:"avgResponseTimeMs"`
	MinResponseTimeMs float64 `json:"minResponseTimeMs"`
	MaxResponseTimeMs float64 `json:"maxResponseTimeMs"`
	P50ResponseMs     float64 `json:"p50ResponseMs"`
	P95ResponseMs     float64 `json:"p95ResponseMs"`
	AvgRPS            float64 `json:"avgRps"`
}

// VisualizationSummaryResponse combines all chart data
type VisualizationSummaryResponse struct {
	TestRunID     string                  `json:"testRunId"`
	Status        string                  `json:"status"`
	Timeseries    []TimeseriesDataPoint   `json:"timeseries"`
	EndpointStats []EndpointStatsResponse `json:"endpointStats"`
	Summary       AggregatedSummary       `json:"summary"`
}

// GraphDataPoint represents minimal data for plotting the main graph
type GraphDataPoint struct {
	Timestamp        int64   `json:"timestamp"`        // Unix milliseconds
	Users            int     `json:"users"`            // Current active users
	RequestsPerSec   float64 `json:"requestsPerSec"`   // RPS
	ErrorsPerSec     float64 `json:"errorsPerSec"`     // Errors per second
	AvgResponseTime  float64 `json:"avgResponseTime"`  // Average response time in seconds
}

// RunGraphResponse returns minimal graph data for plotting
type RunGraphResponse struct {
	RunID      string            `json:"runId"`
	RunName    string            `json:"runName"`
	Status     string            `json:"status"`
	StartedAt  string            `json:"startedAt"`
	DataPoints []GraphDataPoint  `json:"dataPoints"`
}

// RunSummaryResponse returns the 4 key metrics for the summary cards
type RunSummaryResponse struct {
	RunID           string  `json:"runId"`
	RunName         string  `json:"runName"`
	Status          string  `json:"status"`
	StartedAt       string  `json:"startedAt"`
	FinishedAt      string  `json:"finishedAt,omitempty"`
	Duration        string  `json:"duration"`
	TotalRequests   int64   `json:"totalRequests"`
	RequestsPerSec  float64 `json:"requestsPerSec"`
	ErrorRate       float64 `json:"errorRate"`        // Percentage
	AvgResponseTime float64 `json:"avgResponseTime"` // In seconds
	// Test configuration
	TargetUsers     int     `json:"targetUsers"`
	SpawnRate       float64 `json:"spawnRate"`
	DurationSeconds *int    `json:"durationSeconds,omitempty"`
}

// RequestLogEntry represents a single request in the live log
type RequestLogEntry struct {
	Timestamp      int64   `json:"timestamp"`      // Unix milliseconds
	RequestType    string  `json:"requestType"`    // GET, POST, etc.
	ResponseTime   float64 `json:"responseTime"`   // In milliseconds
	URL            string  `json:"url"`
	ResponseLength int64   `json:"responseLength"`
	Success        bool    `json:"success"`
}

// LiveRequestLogResponse returns recent individual requests
type LiveRequestLogResponse struct {
	RunID    string             `json:"runId"`
	Requests []RequestLogEntry  `json:"requests"`
	Total    int                `json:"total"`    // Total requests in the log
	Limit    int                `json:"limit"`    // Number of requests returned
}
