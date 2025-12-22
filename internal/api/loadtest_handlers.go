package api

import (
	"encoding/json"
	"net/http"
	"time"

	"Load-manager-cli/internal/domain"
	"Load-manager-cli/internal/store"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// LoadTest handlers

// CreateLoadTest handles POST /v1/load-tests
func (h *Handler) CreateLoadTest(w http.ResponseWriter, r *http.Request) {
	var req CreateLoadTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	nowMillis := time.Now().UnixMilli()
	test := &domain.LoadTest{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		Tags:               req.Tags,
		AccountID:          req.AccountID,
		OrgID:              req.OrgID,
		ProjectID:          req.ProjectID,
		EnvID:              req.EnvID,
		LocustClusterID:    req.LocustClusterID,
		TargetURL:          req.TargetURL,
		Locustfile:         req.Locustfile,
		ScenarioID:         req.ScenarioID,
		DefaultUsers:       req.DefaultUsers,
		DefaultSpawnRate:   req.DefaultSpawnRate,
		DefaultDurationSec: req.DefaultDurationSec,
		MaxDurationSec:     req.MaxDurationSec,
		RecentRuns:         []domain.RecentRun{},
		CreatedAt:          nowMillis,
		CreatedBy:          req.CreatedBy,
		UpdatedAt:          nowMillis,
		UpdatedBy:          req.CreatedBy,
		Metadata:           req.Metadata,
	}

	if err := h.loadTestStore.Create(test); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create load test", err)
		return
	}

	respondJSON(w, http.StatusCreated, toLoadTestResponse(test))
}

// GetLoadTest handles GET /v1/load-tests/{id}
func (h *Handler) GetLoadTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	test, err := h.loadTestStore.Get(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test not found", err)
		return
	}

	respondJSON(w, http.StatusOK, toLoadTestResponse(test))
}

// ListLoadTests handles GET /v1/load-tests
func (h *Handler) ListLoadTests(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := &store.LoadTestFilter{}

	if accountID := query.Get("accountId"); accountID != "" {
		filter.AccountID = &accountID
	}

	if orgID := query.Get("orgId"); orgID != "" {
		filter.OrgID = &orgID
	}

	if projectID := query.Get("projectId"); projectID != "" {
		filter.ProjectID = &projectID
	}

	if envID := query.Get("envId"); envID != "" {
		filter.EnvID = &envID
	}

	if tags := query["tags"]; len(tags) > 0 {
		filter.Tags = tags
	}

	tests, err := h.loadTestStore.List(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list load tests", err)
		return
	}

	responses := make([]*LoadTestResponse, len(tests))
	for i, test := range tests {
		responses[i] = toLoadTestResponse(test)
	}

	respondJSON(w, http.StatusOK, responses)
}

// UpdateLoadTest handles PUT /v1/load-tests/{id}
func (h *Handler) UpdateLoadTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	test, err := h.loadTestStore.Get(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test not found", err)
		return
	}

	var req UpdateLoadTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Update fields if provided
	if req.Name != "" {
		test.Name = req.Name
	}
	if req.Description != "" {
		test.Description = req.Description
	}
	if req.Tags != nil {
		test.Tags = req.Tags
	}
	if req.TargetURL != "" {
		test.TargetURL = req.TargetURL
	}
	if req.Locustfile != "" {
		test.Locustfile = req.Locustfile
	}
	if req.ScenarioID != "" {
		test.ScenarioID = req.ScenarioID
	}
	if req.DefaultUsers > 0 {
		test.DefaultUsers = req.DefaultUsers
	}
	if req.DefaultSpawnRate > 0 {
		test.DefaultSpawnRate = req.DefaultSpawnRate
	}
	if req.DefaultDurationSec != nil {
		test.DefaultDurationSec = req.DefaultDurationSec
	}
	if req.MaxDurationSec != nil {
		test.MaxDurationSec = req.MaxDurationSec
	}
	if req.Metadata != nil {
		test.Metadata = req.Metadata
	}

	test.UpdatedAt = time.Now().UnixMilli()
	test.UpdatedBy = req.UpdatedBy

	if err := h.loadTestStore.Update(test); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update load test", err)
		return
	}

	respondJSON(w, http.StatusOK, toLoadTestResponse(test))
}

// DeleteLoadTest handles DELETE /v1/load-tests/{id}
func (h *Handler) DeleteLoadTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	if err := h.loadTestStore.Delete(testID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete load test", err)
		return
	}

	respondJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Load test deleted successfully",
	})
}

// LoadTestRun handlers

// CreateLoadTestRun handles POST /v1/load-tests/{id}/runs
func (h *Handler) CreateLoadTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	loadTestID := vars["id"]

	// Get the load test
	loadTest, err := h.loadTestStore.Get(loadTestID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test not found", err)
		return
	}

	var req CreateLoadTestRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Apply defaults from LoadTest, allow overrides
	targetUsers := loadTest.DefaultUsers
	if req.TargetUsers != nil {
		targetUsers = *req.TargetUsers
	}

	spawnRate := loadTest.DefaultSpawnRate
	if req.SpawnRate != nil {
		spawnRate = *req.SpawnRate
	}

	var durationSeconds *int
	if req.DurationSeconds != nil {
		durationSeconds = req.DurationSeconds
	} else if loadTest.DefaultDurationSec != nil {
		durationSeconds = loadTest.DefaultDurationSec
	}

	// Validate against max duration if set
	if loadTest.MaxDurationSec != nil && durationSeconds != nil && *durationSeconds > *loadTest.MaxDurationSec {
		respondError(w, http.StatusBadRequest, "Duration exceeds maximum allowed duration", nil)
		return
	}

	nowMillis := time.Now().UnixMilli()
	run := &domain.LoadTestRun{
		ID:              uuid.New().String(),
		LoadTestID:      loadTestID,
		Name:            req.Name,
		AccountID:       loadTest.AccountID,
		OrgID:           loadTest.OrgID,
		ProjectID:       loadTest.ProjectID,
		EnvID:           loadTest.EnvID,
		TargetUsers:     targetUsers,
		SpawnRate:       spawnRate,
		DurationSeconds: durationSeconds,
		Status:          domain.LoadTestRunStatusPending,
		CreatedAt:       nowMillis,
		CreatedBy:       req.CreatedBy,
		UpdatedAt:       nowMillis,
		UpdatedBy:       req.CreatedBy,
		Metadata:        req.Metadata,
	}

	if err := h.loadTestRunStore.Create(run); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create load test run", err)
		return
	}

	// TODO: Start the actual test via orchestrator
	// h.orchestrator.StartLoadTestRun(run, loadTest)

	respondJSON(w, http.StatusCreated, toLoadTestRunResponse(run))
}

// GetLoadTestRun handles GET /v1/runs/{id}
func (h *Handler) GetLoadTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	run, err := h.loadTestRunStore.Get(runID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test run not found", err)
		return
	}

	respondJSON(w, http.StatusOK, toLoadTestRunResponse(run))
}

// ListLoadTestRuns handles GET /v1/load-tests/{id}/runs and GET /v1/runs
func (h *Handler) ListLoadTestRuns(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	query := r.URL.Query()
	filter := &store.LoadTestRunFilter{}

	// If called from /v1/load-tests/{id}/runs
	if loadTestID := vars["id"]; loadTestID != "" {
		filter.LoadTestID = &loadTestID
	}

	// Query params
	if accountID := query.Get("accountId"); accountID != "" {
		filter.AccountID = &accountID
	}

	if orgID := query.Get("orgId"); orgID != "" {
		filter.OrgID = &orgID
	}

	if projectID := query.Get("projectId"); projectID != "" {
		filter.ProjectID = &projectID
	}

	if envID := query.Get("envId"); envID != "" {
		filter.EnvID = &envID
	}

	if statusStr := query.Get("status"); statusStr != "" {
		status := domain.LoadTestRunStatus(statusStr)
		filter.Status = &status
	}

	runs, err := h.loadTestRunStore.List(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list load test runs", err)
		return
	}

	responses := make([]*LoadTestRunResponse, len(runs))
	for i, run := range runs {
		responses[i] = toLoadTestRunResponse(run)
	}

	respondJSON(w, http.StatusOK, responses)
}

// StopLoadTestRun handles POST /v1/runs/{id}/stop
func (h *Handler) StopLoadTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	run, err := h.loadTestRunStore.Get(runID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test run not found", err)
		return
	}

	if run.Status != domain.LoadTestRunStatusRunning {
		respondError(w, http.StatusBadRequest, "Can only stop running tests", nil)
		return
	}

	run.Status = domain.LoadTestRunStatusStopping
	run.UpdatedAt = time.Now().UnixMilli()

	if err := h.loadTestRunStore.Update(run); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update load test run", err)
		return
	}

	// TODO: Send stop command to Locust via orchestrator
	// h.orchestrator.StopLoadTestRun(runID)

	respondJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Load test run stopping",
	})
}
