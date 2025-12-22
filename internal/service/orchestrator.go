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
	config           *config.Config
	loadTestStore    store.LoadTestRepository
	loadTestRunStore store.LoadTestRunRepository
	metricsStore     *store.MongoMetricsStore
	clients          map[string]locustclient.Client // Map of clusterID -> client
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	pollInterval     time.Duration
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(cfg *config.Config, loadTestStore store.LoadTestRepository, loadTestRunStore store.LoadTestRunRepository, metricsStore *store.MongoMetricsStore) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())

	o := &Orchestrator{
		config:           cfg,
		loadTestStore:    loadTestStore,
		loadTestRunStore: loadTestRunStore,
		metricsStore:     metricsStore,
		clients:          make(map[string]locustclient.Client),
		ctx:              ctx,
		cancel:           cancel,
		pollInterval:     time.Duration(cfg.Orchestrator.MetricsPollIntervalSeconds) * time.Second,
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
func (o *Orchestrator) CreateTestRun(req *CreateTestRunRequest) (*domain.LoadTestRun, error) {
	log.Printf("[Orchestrator] Creating test run: account=%s, org=%s, project=%s, env=%s, users=%d, spawnRate=%.2f",
		req.AccountID, req.OrgID, req.ProjectID, req.EnvID, req.TargetUsers, req.SpawnRate)
	
	// Validate account/org/project and environment
	cluster, err := o.config.GetLocustCluster(req.AccountID, req.OrgID, req.ProjectID, req.EnvID)
	if err != nil {
		log.Printf("[Orchestrator] Failed to resolve cluster: %v", err)
		return nil, fmt.Errorf("failed to resolve cluster: %w", err)
	}
	
	log.Printf("[Orchestrator] Resolved cluster: id=%s, url=%s", cluster.ID, cluster.BaseURL)

	// Create test run entity
	nowMillis := time.Now().UnixMilli()
	run := &domain.LoadTestRun{
		ID:              uuid.New().String(),
		LoadTestID:      req.LoadTestID,
		Name:            req.Name,
		AccountID:       req.AccountID,
		OrgID:           req.OrgID,
		ProjectID:       req.ProjectID,
		EnvID:           req.EnvID,
		TargetUsers:     req.TargetUsers,
		SpawnRate:       req.SpawnRate,
		DurationSeconds: req.DurationSeconds,
		Status:          domain.LoadTestRunStatusPending,
		CreatedAt:       nowMillis,
		CreatedBy:       req.CreatedBy,
		UpdatedAt:       nowMillis,
		UpdatedBy:       req.CreatedBy,
		Metadata:        req.Metadata,
	}

	// Store the test run
	if err := o.loadTestRunStore.Create(run); err != nil {
		return nil, fmt.Errorf("failed to store test run: %w", err)
	}

	// Start the load test on Locust
	client, err := o.getClient(cluster.ID)
	if err != nil {
		run.Status = domain.LoadTestRunStatusFailed
		_ = o.loadTestRunStore.Update(run)
		return nil, fmt.Errorf("failed to get Locust client: %w", err)
	}

	log.Printf("[Orchestrator] Calling Locust swarm API for test %s", run.ID)
	
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	if err := client.Swarm(ctx, req.TargetUsers, req.SpawnRate); err != nil {
		log.Printf("[Orchestrator] Swarm failed for test %s: %v", run.ID, err)
		run.Status = domain.LoadTestRunStatusFailed
		_ = o.loadTestRunStore.Update(run)
		return nil, fmt.Errorf("failed to start swarm on Locust: %w", err)
	}

	log.Printf("[Orchestrator] Swarm succeeded for test %s, updating status to Running", run.ID)
	
	// Update status to Running
	startedAtMillis := time.Now().UnixMilli()
	run.Status = domain.LoadTestRunStatusRunning
	run.StartedAt = startedAtMillis
	run.UpdatedAt = startedAtMillis

	if err := o.loadTestRunStore.Update(run); err != nil {
		return nil, fmt.Errorf("failed to update test run status: %w", err)
	}

	log.Printf("[Orchestrator] Started test run %s for account=%s, org=%s, project=%s, env=%s",
		run.ID, run.AccountID, run.OrgID, run.ProjectID, run.EnvID)
	return run, nil
}

// RegisterExternalTestRun registers a test that was started externally (e.g., from Locust UI)
// This allows the control plane to track and poll metrics for UI-started tests
func (o *Orchestrator) RegisterExternalTestRun(req *RegisterExternalTestRunRequest) (*domain.LoadTestRun, error) {
	log.Printf("[Orchestrator] Registering external test run: account=%s, org=%s, project=%s, env=%s, users=%d",
		req.AccountID, req.OrgID, req.ProjectID, req.EnvID, req.TargetUsers)
	
	// Validate account/org/project and environment
	cluster, err := o.config.GetLocustCluster(req.AccountID, req.OrgID, req.ProjectID, req.EnvID)
	if err != nil {
		log.Printf("[Orchestrator] Failed to resolve cluster for external test: %v", err)
		return nil, fmt.Errorf("failed to resolve cluster: %w", err)
	}
	
	log.Printf("[Orchestrator] Resolved cluster for external test: id=%s, url=%s", cluster.ID, cluster.BaseURL)
	
	// Create test run entity (already running since it was started externally)
	nowMillis := time.Now().UnixMilli()
	run := &domain.LoadTestRun{
		ID:              uuid.New().String(),
		LoadTestID:      "", // External run, no LoadTest reference
		Name:            "External Locust UI Test",
		AccountID:       req.AccountID,
		OrgID:           req.OrgID,
		ProjectID:       req.ProjectID,
		EnvID:           req.EnvID,
		TargetUsers:     req.TargetUsers,
		SpawnRate:       req.SpawnRate,
		DurationSeconds: req.DurationSeconds,
		Status:          domain.LoadTestRunStatusRunning,
		StartedAt:       nowMillis,
		CreatedAt:       nowMillis,
		CreatedBy:       "locust-ui",
		UpdatedAt:       nowMillis,
		UpdatedBy:       "locust-ui",
		Metadata: map[string]any{
			"source": "locust-ui",
			"registeredAt": time.Now().Format("2006-01-02T15:04:05Z07:00"),
		},
	}
	
	// Store the test run
	if err := o.loadTestRunStore.Create(run); err != nil {
		log.Printf("[Orchestrator] Failed to store external test run: %v", err)
		return nil, fmt.Errorf("failed to store test run: %w", err)
	}
	
	log.Printf("[Orchestrator] Registered external test run %s from Locust UI",
		run.ID)
	return run, nil
}

// StopTestRun stops a running load test
func (o *Orchestrator) StopTestRun(runID string) error {
	run, err := o.loadTestRunStore.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	if run.Status != domain.LoadTestRunStatusRunning {
		return fmt.Errorf("test run is not running (current status: %s)", run.Status)
	}

	// Get cluster from config based on account/org/project/env
	cluster, err := o.config.GetLocustCluster(run.AccountID, run.OrgID, run.ProjectID, run.EnvID)
	if err != nil {
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Get Locust client
	client, err := o.getClient(cluster.ID)
	if err != nil {
		return fmt.Errorf("failed to get Locust client: %w", err)
	}

	// Update status to Stopping
	run.Status = domain.LoadTestRunStatusStopping
	run.UpdatedAt = time.Now().UnixMilli()
	if err := o.loadTestRunStore.Update(run); err != nil {
		return fmt.Errorf("failed to update test run status: %w", err)
	}

	// Stop the load test on Locust
	ctx, cancel := context.WithTimeout(o.ctx, 30*time.Second)
	defer cancel()

	if err := client.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop Locust test: %w", err)
	}

	// Mark as finished (will be updated by callback if configured)
	nowMillis := time.Now().UnixMilli()
	run.Status = domain.LoadTestRunStatusFinished
	run.FinishedAt = nowMillis
	run.UpdatedAt = nowMillis

	if err := o.loadTestRunStore.Update(run); err != nil {
		return fmt.Errorf("failed to update test run finish status: %w", err)
	}

	// Update the LoadTest's recent runs if this run has a LoadTestID
	if run.LoadTestID != "" {
		if err := o.updateRecentRuns(run); err != nil {
			log.Printf("Warning: failed to update recent runs for LoadTest %s: %v", run.LoadTestID, err)
		}
	}

	log.Printf("Stopped test run %s", runID)
	return nil
}

// GetTestRun retrieves a test run by ID
func (o *Orchestrator) GetTestRun(runID string) (*domain.LoadTestRun, error) {
	return o.loadTestRunStore.Get(runID)
}

// ListTestRuns lists test runs with optional filtering
func (o *Orchestrator) ListTestRuns(filter *store.LoadTestRunFilter) ([]*domain.LoadTestRun, error) {
	return o.loadTestRunStore.List(filter)
}

// UpdateMetrics updates the metrics for a test run (called by Locust callbacks or poller)
func (o *Orchestrator) UpdateMetrics(runID string, metrics *domain.MetricSnapshot) error {
	run, err := o.loadTestRunStore.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	run.LastMetrics = metrics
	run.UpdatedAt = time.Now().UnixMilli()

	if err := o.loadTestRunStore.Update(run); err != nil {
		return fmt.Errorf("failed to update test run metrics: %w", err)
	}

	return nil
}

// HandleTestStart handles test_start callback from Locust
func (o *Orchestrator) HandleTestStart(runID string) error {
	run, err := o.loadTestRunStore.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	if run.Status == domain.LoadTestRunStatusPending {
		nowMillis := time.Now().UnixMilli()
		run.Status = domain.LoadTestRunStatusRunning
		run.StartedAt = nowMillis
		run.UpdatedAt = nowMillis

		if err := o.loadTestRunStore.Update(run); err != nil {
			return fmt.Errorf("failed to update test run: %w", err)
		}

		log.Printf("Test run %s started (via callback)", runID)
	}

	return nil
}

// HandleTestStop handles test_stop callback from Locust
func (o *Orchestrator) HandleTestStop(runID string, finalMetrics *domain.MetricSnapshot) error {
	run, err := o.loadTestRunStore.Get(runID)
	if err != nil {
		return fmt.Errorf("failed to get test run: %w", err)
	}

	nowMillis := time.Now().UnixMilli()
	run.Status = domain.LoadTestRunStatusFinished
	run.FinishedAt = nowMillis
	run.UpdatedAt = nowMillis
	run.LastMetrics = finalMetrics

	if err := o.loadTestRunStore.Update(run); err != nil {
		return fmt.Errorf("failed to update test run: %w", err)
	}

	// Update the LoadTest's recent runs if this run has a LoadTestID
	if run.LoadTestID != "" {
		if err := o.updateRecentRuns(run); err != nil {
			log.Printf("Warning: failed to update recent runs for LoadTest %s: %v", run.LoadTestID, err)
		}
	}

	log.Printf("Test run %s finished (via callback)", runID)
	return nil
}

// updateRecentRuns updates the LoadTest's RecentRuns array to include the completed run
// and maintains only the 10 most recent runs
func (o *Orchestrator) updateRecentRuns(run *domain.LoadTestRun) error {
	loadTest, err := o.loadTestStore.Get(run.LoadTestID)
	if err != nil {
		return fmt.Errorf("failed to get load test: %w", err)
	}

	// Create a RecentRun entry
	recentRun := domain.RecentRun{
		ID:              run.ID,
		Name:            run.Name,
		Status:          run.Status,
		TargetUsers:     run.TargetUsers,
		SpawnRate:       run.SpawnRate,
		DurationSeconds: run.DurationSeconds,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		CreatedAt:       run.CreatedAt,
		CreatedBy:       run.CreatedBy,
	}

	// Add to the beginning of the array
	loadTest.RecentRuns = append([]domain.RecentRun{recentRun}, loadTest.RecentRuns...)

	// Keep only the 10 most recent runs
	if len(loadTest.RecentRuns) > 10 {
		loadTest.RecentRuns = loadTest.RecentRuns[:10]
	}

	loadTest.UpdatedAt = time.Now().UnixMilli()

	// Update the LoadTest
	if err := o.loadTestStore.Update(loadTest); err != nil {
		return fmt.Errorf("failed to update load test: %w", err)
	}

	log.Printf("Updated recent runs for LoadTest %s, now tracking %d runs", loadTest.ID, len(loadTest.RecentRuns))
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
	status := domain.LoadTestRunStatusRunning
	runs, err := o.loadTestRunStore.List(&store.LoadTestRunFilter{
		Status: &status,
	})
	if err != nil {
		log.Printf("Error listing active runs: %v", err)
		return
	}

	for _, run := range runs {
		// Check if duration has elapsed
		if run.DurationSeconds != nil && run.StartedAt > 0 {
			startedTime := time.UnixMilli(run.StartedAt)
			elapsed := time.Since(startedTime)
			duration := time.Duration(*run.DurationSeconds) * time.Second

			if elapsed >= duration {
				log.Printf("Test run %s has reached duration limit, stopping", run.ID)
				if err := o.StopTestRun(run.ID); err != nil {
					log.Printf("Error stopping test run %s: %v", run.ID, err)
				}
				continue
			}
		}

		// Get cluster from config
		cluster, err := o.config.GetLocustCluster(run.AccountID, run.OrgID, run.ProjectID, run.EnvID)
		if err != nil {
			log.Printf("Error getting cluster for account %s, org %s, project %s, env %s: %v", run.AccountID, run.OrgID, run.ProjectID, run.EnvID, err)
			continue
		}

		// Poll metrics from Locust
		client, err := o.getClient(cluster.ID)
		if err != nil {
			log.Printf("Error getting client for cluster %s: %v", cluster.ID, err)
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

		// Store metrics in time-series collection
		if o.metricsStore != nil {
			storeCtx, storeCancel := context.WithTimeout(o.ctx, 5*time.Second)
			if err := o.metricsStore.StoreMetric(storeCtx, run.ID, run.AccountID, run.OrgID, run.ProjectID, run.EnvID, metrics); err != nil {
				log.Printf("Error storing metrics in time-series for run %s: %v", run.ID, err)
			}
			storeCancel()
		}

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
	LoadTestID      string         `json:"loadTestId"`
	Name            string         `json:"name,omitempty"`
	AccountID       string         `json:"accountId"`
	OrgID           string         `json:"orgId"`
	ProjectID       string         `json:"projectId"`
	EnvID           string         `json:"envId,omitempty"`
	TargetUsers     int            `json:"targetUsers"`
	SpawnRate       float64        `json:"spawnRate"`
	DurationSeconds *int           `json:"durationSeconds,omitempty"`
	CreatedBy       string         `json:"createdBy"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

// RegisterExternalTestRunRequest represents a request to register an externally-started test
type RegisterExternalTestRunRequest struct {
	AccountID       string
	OrgID           string
	ProjectID       string
	EnvID           string
	ScenarioID      string
	TargetUsers     int
	SpawnRate       float64
	DurationSeconds *int
}
