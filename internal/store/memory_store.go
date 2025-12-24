package store

import (
	"Load-manager-cli/internal/domain"
	"fmt"
	"sync"
)

// LoadTestRepository defines the interface for LoadTest persistence operations
type LoadTestRepository interface {
	Create(test *domain.LoadTest) error
	Get(id string) (*domain.LoadTest, error)
	Update(test *domain.LoadTest) error
	List(filter *LoadTestFilter) ([]*domain.LoadTest, error)
	Delete(id string) error
}

// LoadTestFilter provides filtering options for listing load tests
type LoadTestFilter struct {
	AccountID *string
	OrgID     *string
	ProjectID *string
	EnvID     *string
	Tags      []string
	Limit     int
}

// LoadTestRunRepository defines the interface for LoadTestRun persistence operations
type LoadTestRunRepository interface {
	Create(run *domain.LoadTestRun) error
	Get(id string) (*domain.LoadTestRun, error)
	Update(run *domain.LoadTestRun) error
	List(filter *LoadTestRunFilter) ([]*domain.LoadTestRun, error)
	Delete(id string) error
}

// LoadTestRunFilter provides filtering options for listing load test runs
type LoadTestRunFilter struct {
	LoadTestID *string
	AccountID  *string
	OrgID      *string
	ProjectID  *string
	EnvID      *string
	Status     *domain.LoadTestRunStatus
	Limit      int
}

// InMemoryLoadTestStore is an in-memory implementation of LoadTestRepository
// Thread-safe using RWMutex
type InMemoryLoadTestStore struct {
	mu    sync.RWMutex
	tests map[string]*domain.LoadTest
}

// NewInMemoryLoadTestStore creates a new in-memory load test store
func NewInMemoryLoadTestStore() *InMemoryLoadTestStore {
	return &InMemoryLoadTestStore{
		tests: make(map[string]*domain.LoadTest),
	}
}

// InMemoryLoadTestRunStore is an in-memory implementation of LoadTestRunRepository
// Thread-safe using RWMutex
type InMemoryLoadTestRunStore struct {
	mu   sync.RWMutex
	runs map[string]*domain.LoadTestRun
}

// NewInMemoryLoadTestRunStore creates a new in-memory load test run store
func NewInMemoryLoadTestRunStore() *InMemoryLoadTestRunStore {
	return &InMemoryLoadTestRunStore{
		runs: make(map[string]*domain.LoadTestRun),
	}
}

// LoadTest CRUD operations

// Create stores a new load test
func (s *InMemoryLoadTestStore) Create(test *domain.LoadTest) error {
	if test.ID == "" {
		return fmt.Errorf("load test ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.tests[test.ID]; exists {
		return fmt.Errorf("load test with ID %s already exists", test.ID)
	}
	
	s.tests[test.ID] = copyLoadTest(test)
	return nil
}

// Get retrieves a load test by ID
func (s *InMemoryLoadTestStore) Get(id string) (*domain.LoadTest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	test, exists := s.tests[id]
	if !exists {
		return nil, fmt.Errorf("load test with ID %s not found", id)
	}
	
	return copyLoadTest(test), nil
}

// Update updates an existing load test
func (s *InMemoryLoadTestStore) Update(test *domain.LoadTest) error {
	if test.ID == "" {
		return fmt.Errorf("load test ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.tests[test.ID]; !exists {
		return fmt.Errorf("load test with ID %s not found", test.ID)
	}
	
	s.tests[test.ID] = copyLoadTest(test)
	return nil
}

// List retrieves load tests based on filter criteria
func (s *InMemoryLoadTestStore) List(filter *LoadTestFilter) ([]*domain.LoadTest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var results []*domain.LoadTest
	
	for _, test := range s.tests {
		if filter != nil {
			if filter.AccountID != nil && test.AccountID != *filter.AccountID {
				continue
			}
			if filter.OrgID != nil && test.OrgID != *filter.OrgID {
				continue
			}
			if filter.ProjectID != nil && test.ProjectID != *filter.ProjectID {
				continue
			}
			if filter.EnvID != nil && test.EnvID != *filter.EnvID {
				continue
			}
			if len(filter.Tags) > 0 && !hasAnyTag(test.Tags, filter.Tags) {
				continue
			}
		}
		
		results = append(results, copyLoadTest(test))
		
		if filter != nil && filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	
	return results, nil
}

// Delete removes a load test by ID
func (s *InMemoryLoadTestStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.tests[id]; !exists {
		return fmt.Errorf("load test with ID %s not found", id)
	}
	
	delete(s.tests, id)
	return nil
}

// LoadTestRun CRUD operations

// Create stores a new load test run
func (s *InMemoryLoadTestRunStore) Create(run *domain.LoadTestRun) error {
	if run.ID == "" {
		return fmt.Errorf("load test run ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[run.ID]; exists {
		return fmt.Errorf("load test run with ID %s already exists", run.ID)
	}
	
	s.runs[run.ID] = copyLoadTestRun(run)
	return nil
}

// Get retrieves a load test run by ID
func (s *InMemoryLoadTestRunStore) Get(id string) (*domain.LoadTestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	run, exists := s.runs[id]
	if !exists {
		return nil, fmt.Errorf("load test run with ID %s not found", id)
	}
	
	return copyLoadTestRun(run), nil
}

// Update updates an existing load test run
func (s *InMemoryLoadTestRunStore) Update(run *domain.LoadTestRun) error {
	if run.ID == "" {
		return fmt.Errorf("load test run ID cannot be empty")
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[run.ID]; !exists {
		return fmt.Errorf("load test run with ID %s not found", run.ID)
	}
	
	s.runs[run.ID] = copyLoadTestRun(run)
	return nil
}

// List retrieves load test runs based on filter criteria
func (s *InMemoryLoadTestRunStore) List(filter *LoadTestRunFilter) ([]*domain.LoadTestRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var results []*domain.LoadTestRun
	
	for _, run := range s.runs {
		if filter != nil {
			if filter.LoadTestID != nil && run.LoadTestID != *filter.LoadTestID {
				continue
			}
			if filter.AccountID != nil && run.AccountID != *filter.AccountID {
				continue
			}
			if filter.OrgID != nil && run.OrgID != *filter.OrgID {
				continue
			}
			if filter.ProjectID != nil && run.ProjectID != *filter.ProjectID {
				continue
			}
			if filter.EnvID != nil && run.EnvID != *filter.EnvID {
				continue
			}
			if filter.Status != nil && run.Status != *filter.Status {
				continue
			}
		}
		
		results = append(results, copyLoadTestRun(run))
		
		if filter != nil && filter.Limit > 0 && len(results) >= filter.Limit {
			break
		}
	}
	
	return results, nil
}

// Delete removes a load test run by ID
func (s *InMemoryLoadTestRunStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, exists := s.runs[id]; !exists {
		return fmt.Errorf("load test run with ID %s not found", id)
	}
	
	delete(s.runs, id)
	return nil
}

// copyLoadTest creates a deep copy of a LoadTest to prevent external mutations
func copyLoadTest(test *domain.LoadTest) *domain.LoadTest {
	if test == nil {
		return nil
	}
	
	result := &domain.LoadTest{
		ID:                 test.ID,
		Name:               test.Name,
		Description:        test.Description,
		AccountID:          test.AccountID,
		OrgID:              test.OrgID,
		ProjectID:          test.ProjectID,
		EnvID:              test.EnvID,
		LocustClusterID:    test.LocustClusterID,
		TargetURL:          test.TargetURL,
		LatestRevisionID:   test.LatestRevisionID,
		ScenarioID:         test.ScenarioID,
		DefaultUsers:       test.DefaultUsers,
		DefaultSpawnRate:   test.DefaultSpawnRate,
		CreatedAt:          test.CreatedAt,
		CreatedBy:          test.CreatedBy,
		UpdatedAt:          test.UpdatedAt,
		UpdatedBy:          test.UpdatedBy,
	}
	
	if test.Tags != nil {
		result.Tags = make([]string, len(test.Tags))
		copy(result.Tags, test.Tags)
	}
	
	if test.RecentRuns != nil {
		result.RecentRuns = make([]domain.RecentRun, len(test.RecentRuns))
		copy(result.RecentRuns, test.RecentRuns)
	}
	
	if test.DefaultDurationSec != nil {
		val := *test.DefaultDurationSec
		result.DefaultDurationSec = &val
	}
	
	if test.MaxDurationSec != nil {
		val := *test.MaxDurationSec
		result.MaxDurationSec = &val
	}
	
	if test.RecentRuns != nil {
		result.RecentRuns = make([]domain.RecentRun, len(test.RecentRuns))
		copy(result.RecentRuns, test.RecentRuns)
	}
	
	if test.Metadata != nil {
		result.Metadata = make(map[string]any)
		for k, v := range test.Metadata {
			result.Metadata[k] = v
		}
	}
	
	return result
}

// copyLoadTestRun creates a deep copy of a LoadTestRun to prevent external mutations
func copyLoadTestRun(run *domain.LoadTestRun) *domain.LoadTestRun {
	if run == nil {
		return nil
	}
	
	result := &domain.LoadTestRun{
		ID:               run.ID,
		LoadTestID:       run.LoadTestID,
		ScriptRevisionID: run.ScriptRevisionID,
		Name:             run.Name,
		AccountID:        run.AccountID,
		OrgID:            run.OrgID,
		ProjectID:        run.ProjectID,
		EnvID:            run.EnvID,
		TargetUsers:      run.TargetUsers,
		SpawnRate:        run.SpawnRate,
		Status:           run.Status,
		StartedAt:        run.StartedAt,
		FinishedAt:       run.FinishedAt,
		CreatedAt:        run.CreatedAt,
		CreatedBy:        run.CreatedBy,
		UpdatedAt:        run.UpdatedAt,
		UpdatedBy:        run.UpdatedBy,
	}
	
	if run.DurationSeconds != nil {
		val := *run.DurationSeconds
		result.DurationSeconds = &val
	}
	
	if run.Metadata != nil {
		result.Metadata = make(map[string]any)
		for k, v := range run.Metadata {
			result.Metadata[k] = v
		}
	}
	
	if run.LastMetrics != nil {
		result.LastMetrics = copyMetricSnapshot(run.LastMetrics)
	}
	
	return result
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
		MinResponseMs:     metrics.MinResponseMs,
		MaxResponseMs:     metrics.MaxResponseMs,
		AvgResponseMs:     metrics.AvgResponseMs,
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
					AvgResponseTimeMs:  v.AvgResponseTimeMs,
					MinResponseTime:    v.MinResponseTime,
					MinResponseTimeMs:  v.MinResponseTimeMs,
					MaxResponseTime:    v.MaxResponseTime,
					MaxResponseTimeMs:  v.MaxResponseTimeMs,
					MedianResponseTime: v.MedianResponseTime,
					P50ResponseMs:      v.P50ResponseMs,
					P95ResponseMs:      v.P95ResponseMs,
					RequestsPerSec:     v.RequestsPerSec,
				}
			}
		}
	}
	
	return copy
}

// hasAnyTag checks if any of the filter tags exist in the test tags
func hasAnyTag(testTags []string, filterTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range testTags {
		tagSet[tag] = true
	}
	for _, filterTag := range filterTags {
		if tagSet[filterTag] {
			return true
		}
	}
	return false
}
