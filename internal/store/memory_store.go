package store

import (
	"Load-manager-cli/internal/domain"
	"fmt"
	"sync"
)

// TestRunRepository defines the interface for TestRun persistence operations
type TestRunRepository interface {
	Create(run *domain.TestRun) error
	Get(id string) (*domain.TestRun, error)
	Update(run *domain.TestRun) error
	List(filter *TestRunFilter) ([]*domain.TestRun, error)
	Delete(id string) error
}

// TestRunFilter provides filtering options for listing test runs
type TestRunFilter struct {
	TenantID *string
	EnvID    *string
	Status   *domain.TestRunStatus
	Limit    int
}

// InMemoryTestRunStore is an in-memory implementation of TestRunRepository
// Thread-safe using RWMutex
type InMemoryTestRunStore struct {
	mu   sync.RWMutex
	runs map[string]*domain.TestRun
}

// NewInMemoryTestRunStore creates a new in-memory test run store
func NewInMemoryTestRunStore() *InMemoryTestRunStore {
	return &InMemoryTestRunStore{
		runs: make(map[string]*domain.TestRun),
	}
}

// Create stores a new test run
func (s *InMemoryTestRunStore) Create(run *domain.TestRun) error {
	if run.ID == "" {
		return fmt.Errorf("test run ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("test run with ID %s already exists", run.ID)
	}
	
	// Create a copy to avoid external mutations
	s.runs[run.ID] = copyTestRun(run)
	return nil
}

// Get retrieves a test run by ID
func (s *InMemoryTestRunStore) Get(id string) (*domain.TestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	run, exists := s.runs[id]
	if !exists {
		return nil, fmt.Errorf("test run with ID %s not found", id)
	}
	
	// Return a copy to avoid external mutations
	return copyTestRun(run), nil
}

// Update updates an existing test run
func (s *InMemoryTestRunStore) Update(run *domain.TestRun) error {
	if run.ID == "" {
		return fmt.Errorf("test run ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[run.ID]; !exists {
		return fmt.Errorf("test run with ID %s not found", run.ID)
	}
	
	// Store a copy to avoid external mutations
	s.runs[run.ID] = copyTestRun(run)
	return nil
}

// List retrieves test runs based on filter criteria
func (s *InMemoryTestRunStore) List(filter *TestRunFilter) ([]*domain.TestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var results []*domain.TestRun
	
	for _, run := range s.runs {
		// Apply filters
		if filter != nil {
			if filter.TenantID != nil && run.TenantID != *filter.TenantID {
				continue
			}
			if filter.EnvID != nil && run.EnvID != *filter.EnvID {
				continue
			}
			if filter.Status != nil && run.Status != *filter.Status {
				continue
			}
		}
		
		results = append(results, copyTestRun(run))
		
		// Apply limit
		if filter != nil && filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	
	return results, nil
}

// Delete removes a test run by ID
func (s *InMemoryTestRunStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[id]; !exists {
		return fmt.Errorf("test run with ID %s not found", id)
	}
	
	delete(s.runs, id)
	return nil
}

// copyTestRun creates a deep copy of a TestRun to prevent external mutations
func copyTestRun(run *domain.TestRun) *domain.TestRun {
	if run == nil {
		return nil
	}
	
	copy := &domain.TestRun{
		ID:              run.ID,
		TenantID:        run.TenantID,
		EnvID:           run.EnvID,
		LocustClusterID: run.LocustClusterID,
		ScenarioID:      run.ScenarioID,
		TargetUsers:     run.TargetUsers,
		SpawnRate:       run.SpawnRate,
		Status:          run.Status,
	}
	
	if run.DurationSeconds != nil {
		duration := *run.DurationSeconds
		copy.DurationSeconds = &duration
	}
	
	if run.StartedAt != nil {
		startedAt := *run.StartedAt
		copy.StartedAt = &startedAt
	}
	
	if run.FinishedAt != nil {
		finishedAt := *run.FinishedAt
		copy.FinishedAt = &finishedAt
	}
	
	if run.Metadata != nil {
		copy.Metadata = make(map[string]any)
		for k, v := range run.Metadata {
			copy.Metadata[k] = v
		}
	}
	
	if run.LastMetrics != nil {
		copy.LastMetrics = copyMetricSnapshot(run.LastMetrics)
	}
	
	return copy
}

// copyMetricSnapshot creates a copy of a MetricSnapshot
func copyMetricSnapshot(metrics *domain.MetricSnapshot) *domain.MetricSnapshot {
	if metrics == nil {
		return nil
	}
	
	copy := &domain.MetricSnapshot{
		Timestamp:         metrics.Timestamp,
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
		copy.RequestStats = make(map[string]*domain.ReqStat)
		for k, v := range metrics.RequestStats {
			if v != nil {
				copy.RequestStats[k] = &domain.ReqStat{
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
	
	return copy
}
