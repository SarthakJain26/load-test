package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"Load-manager-cli/internal/store"
	"github.com/gorilla/mux"
)

// VisualizationHandler handles visualization API endpoints
type VisualizationHandler struct {
	loadTestRunStore *store.MongoLoadTestRunStore
	metricsStore     *store.MongoMetricsStore
}

// NewVisualizationHandler creates a new visualization handler
func NewVisualizationHandler(loadTestRunStore *store.MongoLoadTestRunStore, metricsStore *store.MongoMetricsStore) *VisualizationHandler {
	return &VisualizationHandler{
		loadTestRunStore: loadTestRunStore,
		metricsStore:     metricsStore,
	}
}

// GetTimeseriesChart godoc
// @Summary Get detailed timeseries metrics
// @Description Returns comprehensive timeseries data for detailed charts and analysis
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Param from query string false "Start time in RFC3339 format"
// @Param to query string false "End time in RFC3339 format"
// @Success 200 {object} TimeseriesChartResponse "Detailed timeseries metrics"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch timeseries data"
// @Router /runs/{id}/metrics/timeseries [get]
func (h *VisualizationHandler) GetTimeseriesChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loadTestRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	loadTestRun, err := h.loadTestRunStore.Get(loadTestRunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	fromTime, toTime := parseTimeRange(r)
	// Convert time.Time to Unix milliseconds
	var fromMillis, toMillis int64
	if !fromTime.IsZero() {
		fromMillis = fromTime.UnixMilli()
	}
	if !toTime.IsZero() {
		toMillis = toTime.UnixMilli()
	}

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, loadTestRunID, fromMillis, toMillis)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	aggMetrics, err := h.metricsStore.GetAggregatedMetrics(ctx, loadTestRunID)
	if err != nil {
		aggMetrics = &store.AggregatedMetrics{}
	}

	dataPoints := make([]TimeseriesDataPoint, len(metrics))
	for i, m := range metrics {
		dataPoints[i] = TimeseriesDataPoint{
			Timestamp:     m.Timestamp,
			TotalRPS:      m.TotalRPS,
			CurrentUsers:  m.CurrentUsers,
			P50ResponseMs: m.P50ResponseMs,
			P95ResponseMs: m.P95ResponseMs,
			P99ResponseMs: m.P99ResponseMs,
			ErrorRate:     m.ErrorRate,
		}
	}

	duration := "N/A"
	if loadTestRun.StartedAt > 0 && loadTestRun.FinishedAt > 0 {
		startTime := time.UnixMilli(loadTestRun.StartedAt)
		endTime := time.UnixMilli(loadTestRun.FinishedAt)
		duration = endTime.Sub(startTime).String()
	}

	response := TimeseriesChartResponse{
		TestRunID:  loadTestRunID,
		DataPoints: dataPoints,
		Summary: AggregatedSummary{
			AvgRPS:           aggMetrics.AvgRPS,
			MaxRPS:           aggMetrics.MaxRPS,
			MinRPS:           aggMetrics.MinRPS,
			AvgP50Latency:    aggMetrics.AvgP50,
			AvgP95Latency:    aggMetrics.AvgP95,
			AvgP99Latency:    aggMetrics.AvgP99,
			MaxP95Latency:    aggMetrics.MaxP95,
			TotalRequests:    aggMetrics.TotalRequests,
			TotalFailures:    aggMetrics.TotalFailures,
			OverallErrorRate: calculateErrorRate(aggMetrics.TotalRequests, aggMetrics.TotalFailures),
			DataPoints:       aggMetrics.DataPoints,
			Duration:         duration,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetScatterPlot godoc
// @Summary Get scatter plot data
// @Description Returns scatter plot data for response time distribution analysis
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Param from query string false "Start time in RFC3339 format"
// @Param to query string false "End time in RFC3339 format"
// @Success 200 {object} ScatterPlotResponse "Scatter plot data points"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch scatter plot data"
// @Router /runs/{id}/metrics/scatter [get]
func (h *VisualizationHandler) GetScatterPlot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loadTestRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	fromTime, toTime := parseTimeRange(r)
	// Convert time.Time to Unix milliseconds
	var fromMillis, toMillis int64
	if !fromTime.IsZero() {
		fromMillis = fromTime.UnixMilli()
	}
	if !toTime.IsZero() {
		toMillis = toTime.UnixMilli()
	}

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, loadTestRunID, fromMillis, toMillis)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var dataPoints []ScatterDataPoint
	endpointsMap := make(map[string]bool)

	for _, m := range metrics {
		for _, stat := range m.RequestStats {
			dataPoints = append(dataPoints, ScatterDataPoint{
				Timestamp:      m.Timestamp,
				Endpoint:       stat.Name,
				Method:         stat.Method,
				ResponseTimeMs: stat.AvgResponseTimeMs,
				Success:        stat.NumFailures == 0,
			})

			endpointsMap[stat.Method+" "+stat.Name] = true
		}
	}

	endpoints := make([]string, 0, len(endpointsMap))
	for endpoint := range endpointsMap {
		endpoints = append(endpoints, endpoint)
	}

	response := ScatterPlotResponse{
		TestRunID:  loadTestRunID,
		DataPoints: dataPoints,
		Endpoints:  endpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAggregatedStats godoc
// @Summary Get aggregated statistics
// @Description Returns aggregated statistics for overall performance analysis
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Param from query string false "Start time in RFC3339 format"
// @Param to query string false "End time in RFC3339 format"
// @Success 200 {object} VisualizationSummaryResponse "Aggregated statistics"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch aggregated stats"
// @Router /runs/{id}/metrics/aggregate [get]
func (h *VisualizationHandler) GetAggregatedStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loadTestRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	loadTestRun, err := h.loadTestRunStore.Get(loadTestRunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, loadTestRunID, 0, 0)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	aggMetrics, err := h.metricsStore.GetAggregatedMetrics(ctx, loadTestRunID)
	if err != nil {
		aggMetrics = &store.AggregatedMetrics{}
	}

	timeseriesPoints := make([]TimeseriesDataPoint, len(metrics))
	for i, m := range metrics {
		timeseriesPoints[i] = TimeseriesDataPoint{
			Timestamp:     m.Timestamp,
			TotalRPS:      m.TotalRPS,
			CurrentUsers:  m.CurrentUsers,
			P50ResponseMs: m.P50ResponseMs,
			P95ResponseMs: m.P95ResponseMs,
			P99ResponseMs: m.P99ResponseMs,
			ErrorRate:     m.ErrorRate,
		}
	}

	endpointStatsMap := make(map[string]*EndpointStatsResponse)

	for _, m := range metrics {
		for _, stat := range m.RequestStats {
			key := stat.Method + ":" + stat.Name

			if existing, ok := endpointStatsMap[key]; ok {
				existing.TotalRequests += stat.NumRequests
				existing.TotalFailures += stat.NumFailures
				existing.AvgResponseTimeMs = (existing.AvgResponseTimeMs + stat.AvgResponseTimeMs) / 2
				if stat.MinResponseTimeMs < existing.MinResponseTimeMs {
					existing.MinResponseTimeMs = stat.MinResponseTimeMs
				}
				if stat.MaxResponseTimeMs > existing.MaxResponseTimeMs {
					existing.MaxResponseTimeMs = stat.MaxResponseTimeMs
				}
				existing.P50ResponseMs = (existing.P50ResponseMs + stat.P50ResponseMs) / 2
				existing.P95ResponseMs = (existing.P95ResponseMs + stat.P95ResponseMs) / 2
				existing.AvgRPS = (existing.AvgRPS + stat.RequestsPerSec) / 2
			} else {
				endpointStatsMap[key] = &EndpointStatsResponse{
					Endpoint:          stat.Name,
					Method:            stat.Method,
					TotalRequests:     stat.NumRequests,
					TotalFailures:     stat.NumFailures,
					AvgResponseTimeMs: stat.AvgResponseTimeMs,
					MinResponseTimeMs: stat.MinResponseTimeMs,
					MaxResponseTimeMs: stat.MaxResponseTimeMs,
					P50ResponseMs:     stat.P50ResponseMs,
					P95ResponseMs:     stat.P95ResponseMs,
					AvgRPS:            stat.RequestsPerSec,
				}
			}
		}
	}

	endpointStats := make([]EndpointStatsResponse, 0, len(endpointStatsMap))
	for _, stat := range endpointStatsMap {
		stat.ErrorRate = calculateErrorRate(stat.TotalRequests, stat.TotalFailures)
		endpointStats = append(endpointStats, *stat)
	}

	duration := "N/A"
	if loadTestRun.StartedAt > 0 && loadTestRun.FinishedAt > 0 {
		startTime := time.UnixMilli(loadTestRun.StartedAt)
		endTime := time.UnixMilli(loadTestRun.FinishedAt)
		duration = endTime.Sub(startTime).String()
	}

	response := VisualizationSummaryResponse{
		TestRunID:     loadTestRunID,
		Status:        string(loadTestRun.Status),
		Timeseries:    timeseriesPoints,
		EndpointStats: endpointStats,
		Summary: AggregatedSummary{
			AvgRPS:           aggMetrics.AvgRPS,
			MaxRPS:           aggMetrics.MaxRPS,
			MinRPS:           aggMetrics.MinRPS,
			AvgP50Latency:    aggMetrics.AvgP50,
			AvgP95Latency:    aggMetrics.AvgP95,
			AvgP99Latency:    aggMetrics.AvgP99,
			MaxP95Latency:    aggMetrics.MaxP95,
			TotalRequests:    aggMetrics.TotalRequests,
			TotalFailures:    aggMetrics.TotalFailures,
			OverallErrorRate: calculateErrorRate(aggMetrics.TotalRequests, aggMetrics.TotalFailures),
			DataPoints:       aggMetrics.DataPoints,
			Duration:         duration,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseTimeRange(r *http.Request) (time.Time, time.Time) {
	var fromTime, toTime time.Time

	if from := r.URL.Query().Get("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			fromTime = t
		}
	}

	if to := r.URL.Query().Get("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			toTime = t
		}
	}

	return fromTime, toTime
}

func calculateErrorRate(total, failures int64) float64 {
	if total == 0 {
		return 0.0
	}
	return (float64(failures) / float64(total)) * 100.0
}

// GetRunGraph godoc
// @Summary Get run graph data
// @Description Returns minimal graph data optimized for dashboard charts (RPS and response time over time)
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Param from query string false "Start time in RFC3339 format"
// @Param to query string false "End time in RFC3339 format"
// @Success 200 {object} RunGraphResponse "Graph data for visualization"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch graph data"
// @Router /runs/{id}/graph [get]
func (h *VisualizationHandler) GetRunGraph(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Get the run details
	run, err := h.loadTestRunStore.Get(runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Get time range (optional filters)
	fromTime, toTime := parseTimeRange(r)
	var fromMillis, toMillis int64
	if !fromTime.IsZero() {
		fromMillis = fromTime.UnixMilli()
	}
	if !toTime.IsZero() {
		toMillis = toTime.UnixMilli()
	}

	// Fetch metrics from store
	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, runID, fromMillis, toMillis)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to minimal graph data points
	dataPoints := make([]GraphDataPoint, len(metrics))
	for i, m := range metrics {
		// Calculate errors per second from error rate and total RPS
		errorsPerSec := (m.ErrorRate / 100.0) * m.TotalRPS
		
		// Convert response time from ms to seconds
		avgResponseTimeSec := m.P50ResponseMs / 1000.0

		dataPoints[i] = GraphDataPoint{
			Timestamp:       m.Timestamp.UnixMilli(),
			Users:           m.CurrentUsers,
			RequestsPerSec:  m.TotalRPS,
			ErrorsPerSec:    errorsPerSec,
			AvgResponseTime: avgResponseTimeSec,
		}
	}

	// Format started time
	startedAt := ""
	if run.StartedAt > 0 {
		startedAt = time.UnixMilli(run.StartedAt).Format(time.RFC3339)
	}

	response := RunGraphResponse{
		RunID:      runID,
		RunName:    run.Name,
		Status:     string(run.Status),
		StartedAt:  startedAt,
		DataPoints: dataPoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetRunSummary godoc
// @Summary Get run summary metrics
// @Description Returns the 4 key metrics for dashboard cards (total requests, RPS, avg response time, error rate)
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Success 200 {object} RunSummaryResponse "Summary metrics"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch summary"
// @Router /runs/{id}/summary [get]
func (h *VisualizationHandler) GetRunSummary(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Get the run details
	run, err := h.loadTestRunStore.Get(runID)
	if err != nil {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	// Get aggregated metrics
	aggMetrics, err := h.metricsStore.GetAggregatedMetrics(ctx, runID)
	if err != nil {
		// Return partial response if metrics not available
		aggMetrics = &store.AggregatedMetrics{}
	}

	// Calculate duration
	duration := "Running..."
	if run.StartedAt > 0 && run.FinishedAt > 0 {
		startTime := time.UnixMilli(run.StartedAt)
		endTime := time.UnixMilli(run.FinishedAt)
		duration = endTime.Sub(startTime).Round(time.Second).String()
	} else if run.StartedAt > 0 {
		// Calculate elapsed time for running tests
		startTime := time.UnixMilli(run.StartedAt)
		duration = time.Since(startTime).Round(time.Second).String()
	}

	// Format timestamps
	startedAt := ""
	if run.StartedAt > 0 {
		startedAt = time.UnixMilli(run.StartedAt).Format(time.RFC3339)
	}
	finishedAt := ""
	if run.FinishedAt > 0 {
		finishedAt = time.UnixMilli(run.FinishedAt).Format(time.RFC3339)
	}

	// Calculate average response time in seconds
	avgResponseTime := aggMetrics.AvgP50 / 1000.0

	response := RunSummaryResponse{
		RunID:           runID,
		RunName:         run.Name,
		Status:          string(run.Status),
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		Duration:        duration,
		TotalRequests:   aggMetrics.TotalRequests,
		RequestsPerSec:  aggMetrics.AvgRPS,
		ErrorRate:       calculateErrorRate(aggMetrics.TotalRequests, aggMetrics.TotalFailures),
		AvgResponseTime: avgResponseTime,
		TargetUsers:     run.TargetUsers,
		SpawnRate:       run.SpawnRate,
		DurationSeconds: run.DurationSeconds,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetLiveRequestLog godoc
// @Summary Get live request statistics
// @Description Returns endpoint statistics in a log-like format showing performance per endpoint
// @Tags Visualization
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Param limit query int false "Maximum number of entries to return" default(100)
// @Success 200 {array} RequestLogEntry "List of request statistics per endpoint"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to fetch request log"
// @Router /runs/{id}/requests [get]
func (h *VisualizationHandler) GetLiveRequestLog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Parse limit parameter (default 100, max 500)
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := json.Number(limitStr).Int64(); err == nil && parsedLimit > 0 {
			limit = int(parsedLimit)
			if limit > 500 {
				limit = 500
			}
		}
	}

	// Get recent metrics (last few data points)
	fromTime, toTime := parseTimeRange(r)
	var fromMillis, toMillis int64
	if !fromTime.IsZero() {
		fromMillis = fromTime.UnixMilli()
	}
	if !toTime.IsZero() {
		toMillis = toTime.UnixMilli()
	}

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, runID, fromMillis, toMillis)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Collect endpoint data from recent metrics as log entries
	var logEntries []RequestLogEntry
	
	// Take the most recent metrics first (reverse order)
	for i := len(metrics) - 1; i >= 0 && len(logEntries) < limit; i-- {
		m := metrics[i]
		
		for _, stat := range m.RequestStats {
			if len(logEntries) >= limit {
				break
			}
			
			// Create a log entry for each endpoint stat
			// Note: This is aggregated data, not individual requests
			logEntries = append(logEntries, RequestLogEntry{
				Timestamp:      m.Timestamp.UnixMilli(),
				RequestType:    stat.Method,
				ResponseTime:   stat.AvgResponseTimeMs,
				URL:            stat.Name,
				ResponseLength: 0, // Content length not tracked in aggregated stats
				Success:        stat.NumFailures == 0,
			})
		}
	}

	response := LiveRequestLogResponse{
		RunID:    runID,
		Requests: logEntries,
		Total:    len(logEntries),
		Limit:    limit,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
