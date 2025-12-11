package locustclient

import (
	"Load-manager-cli/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	
	log.Printf("[Locust Client] Starting swarm: users=%d, spawnRate=%.2f, endpoint=%s", users, spawnRate, endpoint)
	
	// Locust /swarm endpoint expects form-encoded data
	formData := url.Values{}
	formData.Set("user_count", strconv.Itoa(users))
	formData.Set("spawn_rate", fmt.Sprintf("%.2f", spawnRate))
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create swarm request: %w", err)
	}
	
	// Set proper content type for form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	if c.authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.authToken))
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[Locust Client] Swarm request failed: %v", err)
		return fmt.Errorf("failed to execute swarm request: %w", err)
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("[Locust Client] Swarm failed with status %d: %s", resp.StatusCode, string(body))
		return fmt.Errorf("swarm request failed with status %d: %s", resp.StatusCode, string(body))
	}
	
	log.Printf("[Locust Client] Swarm successful. Response: %s", string(body))
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
	
	// Read body for both parsing and debugging
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var statsResponse LocustStatsResponse
	if err := json.Unmarshal(body, &statsResponse); err != nil {
		return nil, fmt.Errorf("failed to decode stats response: %w", err)
	}

	return convertToMetricSnapshot(&statsResponse), nil
}

// LocustStatsResponse represents the response from Locust /stats/requests endpoint
// Note: Field names are based on Locust's actual API response format
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
		CurrentFailPerSec  float64 `json:"current_fail_per_sec"`
	} `json:"stats"`
	// Top-level aggregated fields
	TotalRps          float64 `json:"total_rps"`           // May also be "current_rps_total"
	FailRatio         float64 `json:"fail_ratio"`          // Decimal 0-1, not percentage
	CurrentUserCount  int     `json:"user_count"`          // Active users
	State             string  `json:"state"`               // "running", "stopped", etc.
	TotalAvgResponseTime float64 `json:"total_avg_response_time"` // Average across all requests
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
	var numValidStats int

	// Aggregate stats from individual endpoints
	for _, stat := range stats.Stats {
		// Skip the "Aggregated" or "Total" entry if present
		if stat.Name == "Aggregated" || stat.Name == "Total" {
			// But use it for aggregate metrics if available
			if stat.NumRequests > 0 {
				snapshot.TotalRequests = stat.NumRequests
				snapshot.TotalFailures = stat.NumFailures
				snapshot.AverageResponseMs = stat.AvgResponseTime
				snapshot.P50ResponseMs = stat.MedianResponseTime
			}
			continue
		}

		// Accumulate for our own aggregation (fallback if no Aggregated entry)
		totalRequests += stat.NumRequests
		totalFailures += stat.NumFailures

		if stat.NumRequests > 0 {
			sumAvgResponseTime += stat.AvgResponseTime
			numValidStats++

			// Track highest median for percentile approximation
			if stat.MedianResponseTime > p50 {
				p50 = stat.MedianResponseTime
			}
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

	// Use our aggregation if we didn't get it from the "Aggregated" entry
	if snapshot.TotalRequests == 0 {
		snapshot.TotalRequests = totalRequests
		snapshot.TotalFailures = totalFailures
	}

	if snapshot.AverageResponseMs == 0 && numValidStats > 0 {
		snapshot.AverageResponseMs = sumAvgResponseTime / float64(numValidStats)
	}

	if snapshot.P50ResponseMs == 0 {
		snapshot.P50ResponseMs = p50
	}

	// Approximations for P95 and P99 if not provided
	if snapshot.P50ResponseMs > 0 {
		snapshot.P95ResponseMs = snapshot.P50ResponseMs * 1.5
		snapshot.P99ResponseMs = snapshot.P50ResponseMs * 2.0
	}

	// Use TotalAvgResponseTime if available
	if stats.TotalAvgResponseTime > 0 {
		snapshot.AverageResponseMs = stats.TotalAvgResponseTime
	}
	
	return snapshot
}
