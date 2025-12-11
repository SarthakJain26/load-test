package service

import (
	"Load-manager-cli/internal/config"
	"Load-manager-cli/internal/domain"
	"Load-manager-cli/internal/locustclient"
	"Load-manager-cli/internal/store"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Orchestrator manages the lifecycle of load test runs and coordinates with Locust clusters
type Orchestrator struct {
	config       *config.Config
	store        store.TestRunRepository
	clients      map[string]locustclient.Client // Map of clusterID -> client
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	pollInterval time.Duration
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(cfg *config.Config, store store.TestRunRepository) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	o := &Orchestrator{
		config:       cfg,
		store:        store,
		clients:      make(map[string]locustclient.Client),
		ctx:          ctx,
		cancel:       cancel,
		pollInterval: time.Duration(cfg.Orchestrator.MetricsPollIntervalSeconds) * time.Second,
	}

	// Initialize Locust clients for each configured cluster
	for _, clusterCfg := range cfg.LocustClusters {
		client := locustclient.NewHTTPClient(clusterCfg.BaseURL, clusterCfg.AuthToken)
		o.clients[clusterCfg.ID] = client
	}

	return o
}

// Start begins the orchestrator background tasks (metrics polling, duration checks, etc.)
func (o *Orchestrator) Start() {
	go o.runMetricsPoller()
	log.Println("Orchestrator started")
}

// Stop gracefully shuts down the orchestrator
func (o *Orchestrator) Stop() {
	o.cancel()
	log.Println("Orchestrator stopped")
}

// CreateTestRun creates and starts a new load test run
func (o *Orchestrator) CreateTestRun(req *CreateTestRunRequest) (*domain.TestRun, error) {
	log.Printf("[Orchestrator] Creating test run: tenant=%s, env=%s, users=%d, spawnRate=%.2f",
		req.TenantID, req.EnvID, req.TargetUsers, req.SpawnRate)
	
	// Validate tenant and environment
	cluster, err := o.config.GetLocustCluster(req.TenantID, req.EnvID)
	if err != nil {
		log.Printf("[Orchestrator] Failed to resolve cluster: %v", err)
		return nil, fmt.Errorf("failed to resolve cluster: %w", err)
	}
	
	log.Printf("[Orchestrator] Resolved cluster: id=%s, url=%s", cluster.ID, cluster.BaseURL)

	// Create test run entity
	run := &domain.TestRun{
		ID:              uuid.New().String(),
		TenantID:        req.TenantID,
		EnvID:           req.EnvID,
		LocustClusterID: cluster.ID,
		ScenarioID:      req.ScenarioID,
		TargetUsers:     req.TargetUsers,
		SpawnRate:       req.SpawnRate,
		DurationSeconds: req.DurationSeconds,
		Status:          domain.TestRunStatusPending,
		Metadata:        req.Metadata,
	}

	// Store the test run
	if err := o.store.Create(run); err != nil {
		return nil, fmt.Errorf("failed to store test run: %w", err)
	}

	// Start the load test on Locust
	client, err := o.getClient(cluster.ID)
	if err != nil {
		run.Status = domain.TestRunStatusFailed
		_ = o.store.Update(run)
		return nil, fmt.Errorf("failed to get Locust client: %w", err)
	}

	log.Printf("[Orchestrator] Calling Locust swarm API for test %s", run.ID)
	
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	if err := client.Swarm(ctx, req.TargetUsers, req.SpawnRate); err != nil {
		log.Printf("[Orchestrator] Swarm failed for test %s: %v", run.ID, err)
		run.Status = domain.TestRunStatusFailed
		_ = o.store.Update(run)
		return nil, fmt.Errorf("failed to start swarm on Locust: %w", err)
	}

	log.Printf("[Orchestrator] Swarm succeeded for test %s, updating status to Running", run.ID)
	
	// Update status to Running
	now := time.Now()
	run.Status = domain.TestRunStatusRunning
	run.StartedAt = &now

	if err := o.store.Update(run); err != nil {
		return nil, fmt.Errorf("failed to update test run status: %w", err)
	}

	log.Printf("[Orchestrator] Started test run %s for tenant=%s, env=%s on cluster %s",
		run.ID, run.TenantID, run.EnvID, cluster.ID)
	return run, nil
}

// StopTestRun stops a running load test
func (o *Orchestrator) StopTestRun(runID string) error {
	run, err := o.store.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	if run.Status != domain.TestRunStatusRunning {
		return fmt.Errorf("test run is not running (current status: %s)", run.Status)
	}

	// Get Locust client
	client, err := o.getClient(run.LocustClusterID)
	if err != nil {
		return fmt.Errorf("failed to get Locust client: %w", err)
	}

	// Update status to Stopping
	run.Status = domain.TestRunStatusStopping
	if err := o.store.Update(run); err != nil {
		return fmt.Errorf("failed to update test run status: %w", err)
	}

	// Stop the load test on Locust
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	if err := client.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop Locust test: %w", err)
	}

	// Mark as finished (will be updated by callback if configured)
	now := time.Now()
	run.Status = domain.TestRunStatusFinished
	run.FinishedAt = &now

	if err := o.store.Update(run); err != nil {
		return fmt.Errorf("failed to update test run finish status: %w", err)
	}

	log.Printf("Stopped test run %s", runID)
	return nil
}

// GetTestRun retrieves a test run by ID
func (o *Orchestrator) GetTestRun(runID string) (*domain.TestRun, error) {
	return o.store.Get(runID)
}

// ListTestRuns lists test runs with optional filtering
func (o *Orchestrator) ListTestRuns(filter *store.TestRunFilter) ([]*domain.TestRun, error) {
	return o.store.List(filter)
}

// UpdateMetrics updates the metrics for a test run (called by Locust callbacks or poller)
func (o *Orchestrator) UpdateMetrics(runID string, metrics *domain.MetricSnapshot) error {
	run, err := o.store.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	run.LastMetrics = metrics

	if err := o.store.Update(run); err != nil {
		return fmt.Errorf("failed to update test run metrics: %w", err)
	}

	return nil
}

// HandleTestStart handles test_start callback from Locust
func (o *Orchestrator) HandleTestStart(runID string) error {
	run, err := o.store.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	if run.Status == domain.TestRunStatusPending {
		now := time.Now()
		run.Status = domain.TestRunStatusRunning
		run.StartedAt = &now

		if err := o.store.Update(run); err != nil {
			return fmt.Errorf("failed to update test run: %w", err)
		}

		log.Printf("Test run %s started (via callback)", runID)
	}

	return nil
}

// HandleTestStop handles test_stop callback from Locust
func (o *Orchestrator) HandleTestStop(runID string, finalMetrics *domain.MetricSnapshot) error {
	run, err := o.store.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	now := time.Now()
	run.Status = domain.TestRunStatusFinished
	run.FinishedAt = &now
	run.LastMetrics = finalMetrics

	if err := o.store.Update(run); err != nil {
		return fmt.Errorf("failed to update test run: %w", err)
	}

	log.Printf("Test run %s finished (via callback)", runID)
	return nil
}

// runMetricsPoller periodically polls Locust clusters for metrics on active runs
func (o *Orchestrator) runMetricsPoller() {
	ticker := time.NewTicker(o.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.pollMetrics()
		}
	}
}

// pollMetrics polls all active test runs for updated metrics
func (o *Orchestrator) pollMetrics() {
	// Get all running tests
	status := domain.TestRunStatusRunning
	runs, err := o.store.List(&store.TestRunFilter{
		Status: &status,
	})
	if err != nil {
		log.Printf("Error listing active runs: %v", err)
		return
	}

	for _, run := range runs {
		// Check if duration has elapsed
		if run.DurationSeconds != nil && run.StartedAt != nil {
			elapsed := time.Since(*run.StartedAt)
			duration := time.Duration(*run.DurationSeconds) * time.Second

			if elapsed >= duration {
				log.Printf("Test run %s has reached duration limit, stopping", run.ID)
				if err := o.StopTestRun(run.ID); err != nil {
					log.Printf("Error stopping test run %s: %v", run.ID, err)
				}
				continue
			}
		}

		// Poll metrics from Locust
		client, err := o.getClient(run.LocustClusterID)
		if err != nil {
			log.Printf("Error getting client for cluster %s: %v", run.LocustClusterID, err)
			continue
		}

		ctx, cancel := context.WithTimeout(o.ctx, 10*time.Second)
		metrics, err := client.GetStats(ctx)
		cancel()

		if err != nil {
			log.Printf("Error fetching stats for run %s: %v", run.ID, err)
			continue
		}

		// Log metrics summary for debugging
		log.Printf("Polled metrics for run %s: RPS=%.2f, Requests=%d, Failures=%d, Users=%d",
			run.ID, metrics.TotalRPS, metrics.TotalRequests, metrics.TotalFailures, metrics.CurrentUsers)

		// Update metrics
		if err := o.UpdateMetrics(run.ID, metrics); err != nil {
			log.Printf("Error updating metrics for run %s: %v", run.ID, err)
		}
	}
}

// getClient retrieves a Locust client for the given cluster ID
func (o *Orchestrator) getClient(clusterID string) (locustclient.Client, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	client, exists := o.clients[clusterID]
	if !exists {
		return nil, fmt.Errorf("no client found for cluster %s", clusterID)
	}

	return client, nil
}

// CreateTestRunRequest represents a request to create and start a test run
type CreateTestRunRequest struct {
	TenantID        string         `json:"tenantId"`
	EnvID           string         `json:"envId"`
	ScenarioID      string         `json:"scenarioId"`
	TargetUsers     int            `json:"targetUsers"`
	SpawnRate       float64        `json:"spawnRate"`
	DurationSeconds *int           `json:"durationSeconds,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}
