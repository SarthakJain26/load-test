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
