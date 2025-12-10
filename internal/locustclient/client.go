package locustclient

import (
	"Load-manager-cli/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is an interface for interacting with Locust master HTTP API
type Client interface {
	Swarm(ctx context.Context, users int, spawnRate float64) error
	Stop(ctx context.Context) error
	GetStats(ctx context.Context) (*domain.MetricSnapshot, error)
}

// HTTPClient implements the Client interface using HTTP calls to Locust master
type HTTPClient struct {
	baseURL    string
	authToken  string
	httpClient *http.Client
}

// NewHTTPClient creates a new Locust HTTP client
func NewHTTPClient(baseURL, authToken string) *HTTPClient {
	return &HTTPClient{
		baseURL:   baseURL,
		authToken: authToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Swarm starts a load test with specified users and spawn rate
// Calls the /swarm endpoint on Locust master
func (c *HTTPClient) Swarm(ctx context.Context, users int, spawnRate float64) error {
	endpoint := fmt.Sprintf("%s/swarm", c.baseURL)
	
	// Locust /swarm endpoint accepts form data or query params
	params := url.Values{}
	params.Set("user_count", strconv.Itoa(users))
	params.Set("spawn_rate", fmt.Sprintf("%.2f", spawnRate))
	
	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())
	
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create swarm request: %w", err)
	}
	
	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute swarm request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("swarm request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// Stop stops the current load test
// Calls the /stop endpoint on Locust master
func (c *HTTPClient) Stop(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/stop", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create stop request: %w", err)
	}
	
	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute stop request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	return nil
}

// GetStats retrieves current statistics from Locust master
// Calls the /stats/requests endpoint
func (c *HTTPClient) GetStats(ctx context.Context) (*domain.MetricSnapshot, error) {
	endpoint := fmt.Sprintf("%s/stats/requests", c.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create stats request: %w", err)
	}
	
	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute stats request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stats request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	var statsResponse LocustStatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&statsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode stats response: %w", err)
	}
	
	return convertToMetricSnapshot(&statsResponse), nil
}

// LocustStatsResponse represents the response from Locust /stats/requests endpoint
type LocustStatsResponse struct {
	Stats []struct {
		Method             string  `json:"method"`
		Name               string  `json:"name"`
		NumRequests        int64   `json:"num_requests"`
		NumFailures        int64   `json:"num_failures"`
		AvgResponseTime    float64 `json:"avg_response_time"`
		MinResponseTime    float64 `json:"min_response_time"`
		MaxResponseTime    float64 `json:"max_response_time"`
		MedianResponseTime float64 `json:"median_response_time"`
		CurrentRps         float64 `json:"current_rps"`
	} `json:"stats"`
	TotalRps          float64 `json:"total_rps"`
	FailRatio         float64 `json:"fail_ratio"`
	CurrentUserCount  int     `json:"current_user_count"`
	State             string  `json:"state"`
	UserCount         int     `json:"user_count"`
}

// convertToMetricSnapshot converts Locust stats response to our domain MetricSnapshot
func convertToMetricSnapshot(stats *LocustStatsResponse) *domain.MetricSnapshot {
	snapshot := &domain.MetricSnapshot{
		Timestamp:     time.Now(),
		TotalRPS:      stats.TotalRps,
		ErrorRate:     stats.FailRatio * 100, // Convert to percentage
		CurrentUsers:  stats.CurrentUserCount,
		RequestStats:  make(map[string]*domain.ReqStat),
	}
	
	var totalRequests, totalFailures int64
	var sumAvgResponseTime float64
	var p50 float64
	
	// Aggregate stats from individual endpoints
	for _, stat := range stats.Stats {
		// Skip the "Aggregated" entry if present
		if stat.Name == "Aggregated" || stat.Name == "Total" {
			continue
		}
		
		totalRequests += stat.NumRequests
		totalFailures += stat.NumFailures
		sumAvgResponseTime += stat.AvgResponseTime
		
		// Use median as approximation for percentiles
		// In a real implementation, we'd need more detailed percentile data from Locust
		if stat.MedianResponseTime > p50 {
			p50 = stat.MedianResponseTime
		}
		
		// Store per-request stats
		key := fmt.Sprintf("%s_%s", stat.Method, stat.Name)
		snapshot.RequestStats[key] = &domain.ReqStat{
			Method:             stat.Method,
			Name:               stat.Name,
			NumRequests:        stat.NumRequests,
			NumFailures:        stat.NumFailures,
			AvgResponseTime:    stat.AvgResponseTime,
			MinResponseTime:    stat.MinResponseTime,
			MaxResponseTime:    stat.MaxResponseTime,
			MedianResponseTime: stat.MedianResponseTime,
			RequestsPerSec:     stat.CurrentRps,
		}
	}
	
	snapshot.TotalRequests = totalRequests
	snapshot.TotalFailures = totalFailures
	
	if len(stats.Stats) > 0 {
		snapshot.AverageResponseMs = sumAvgResponseTime / float64(len(stats.Stats))
	}
	
	// For now, use median as approximation for percentiles
	// Locust's detailed percentile data would need custom parsing
	snapshot.P50ResponseMs = p50
	snapshot.P95ResponseMs = p50 * 1.5 // Rough approximation
	snapshot.P99ResponseMs = p50 * 2.0 // Rough approximation
	
	return snapshot
}
