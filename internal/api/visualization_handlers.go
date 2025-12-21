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
	testRunStore *store.MongoTestRunStore
	metricsStore *store.MongoMetricsStore
}

// NewVisualizationHandler creates a new visualization handler
func NewVisualizationHandler(testRunStore *store.MongoTestRunStore, metricsStore *store.MongoMetricsStore) *VisualizationHandler {
	return &VisualizationHandler{
		testRunStore: testRunStore,
		metricsStore: metricsStore,
	}
}

// GetTimeseriesChart returns time-series data for line charts
func (h *VisualizationHandler) GetTimeseriesChart(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	testRun, err := h.testRunStore.GetByID(ctx, testRunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	fromTime, toTime := parseTimeRange(r)

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, testRunID, fromTime, toTime)
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	aggMetrics, err := h.metricsStore.GetAggregatedMetrics(ctx, testRunID)
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
	if testRun.StartedAt != nil && testRun.FinishedAt != nil {
		duration = testRun.FinishedAt.Sub(*testRun.StartedAt).String()
	}

	response := TimeseriesChartResponse{
		TestRunID:  testRunID,
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

// GetScatterPlot returns scatter plot data (response time distribution)
func (h *VisualizationHandler) GetScatterPlot(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	fromTime, toTime := parseTimeRange(r)

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, testRunID, fromTime, toTime)
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
		TestRunID:  testRunID,
		DataPoints: dataPoints,
		Endpoints:  endpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetAggregatedStats returns aggregated statistics
func (h *VisualizationHandler) GetAggregatedStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testRunID := vars["id"]

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	testRun, err := h.testRunStore.GetByID(ctx, testRunID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	metrics, err := h.metricsStore.GetMetricsTimeseries(ctx, testRunID, time.Time{}, time.Time{})
	if err != nil {
		http.Error(w, "Failed to fetch metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	aggMetrics, err := h.metricsStore.GetAggregatedMetrics(ctx, testRunID)
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
	if testRun.StartedAt != nil && testRun.FinishedAt != nil {
		duration = testRun.FinishedAt.Sub(*testRun.StartedAt).String()
	}

	response := VisualizationSummaryResponse{
		TestRunID:     testRunID,
		Status:        string(testRun.Status),
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
