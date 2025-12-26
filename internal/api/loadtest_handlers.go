package api

import (
	"Load-manager-cli/internal/scriptprocessor"
	"Load-manager-cli/internal/service"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"Load-manager-cli/internal/domain"
	"Load-manager-cli/internal/store"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// LoadTest handlers

// CreateLoadTest godoc
// @Summary Create a new load test
// @Description Creates a new load test configuration with an initial script revision
// @Tags LoadTests
// @Accept json
// @Produce json
// @Param request body CreateLoadTestRequest true "Load test configuration with base64 encoded script"
// @Success 201 {object} LoadTestResponse "Load test created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 500 {object} ErrorResponse "Failed to create load test"
// @Router /load-tests [post]
func (h *Handler) CreateLoadTest(w http.ResponseWriter, r *http.Request) {
	var req CreateLoadTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	nowMillis := time.Now().UnixMilli()
	testID := uuid.New().String()

	// Automatically inject Harness plugin import into user script
	log.Printf("[LoadTest] Injecting Harness plugin into script for test %s", testID)
	enhancedScript, err := scriptprocessor.InjectHarnessPluginBase64(req.ScriptContent)
	if err != nil {
		log.Printf("[LoadTest] Failed to inject plugin: %v", err)
		respondError(w, http.StatusBadRequest, "Failed to process script", err)
		return
	}
	log.Printf("[LoadTest] Plugin injection successful for test %s", testID)

	// Create initial script revision with enhanced script
	revisionID := uuid.New().String()
	revision := &domain.ScriptRevision{
		ID:             revisionID,
		LoadTestID:     testID,
		RevisionNumber: 1,
		ScriptContent:  enhancedScript, // Store enhanced script with plugin import
		Description:    "Initial version",
		CreatedAt:      nowMillis,
		CreatedBy:      req.CreatedBy,
	}

	if err := h.scriptRevisionStore.Create(revision); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create script revision", err)
		return
	}

	// Create load test with reference to the revision
	test := &domain.LoadTest{
		ID:                 testID,
		Name:               req.Name,
		Description:        req.Description,
		Tags:               req.Tags,
		AccountID:          req.AccountID,
		OrgID:              req.OrgID,
		ProjectID:          req.ProjectID,
		EnvID:              req.EnvID,
		LocustClusterID:    req.LocustClusterID,
		TargetURL:          req.TargetURL,
		LatestRevisionID:   revisionID,
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

// GetLoadTest godoc
// @Summary Get load test by ID
// @Description Retrieves a specific load test configuration by its ID, including the user's original script (without plugin)
// @Tags LoadTests
// @Produce json
// @Param id path string true "Load Test ID"
// @Success 200 {object} LoadTestResponse "Load test details with script content"
// @Failure 404 {object} ErrorResponse "Load test not found"
// @Router /load-tests/{id} [get]
func (h *Handler) GetLoadTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	test, err := h.loadTestStore.Get(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test not found", err)
		return
	}

	// Fetch the latest script revision and strip plugin import
	var cleanScriptContent string
	if test.LatestRevisionID != "" {
		revision, err := h.scriptRevisionStore.Get(test.LatestRevisionID)
		if err != nil {
			log.Printf("[GetLoadTest] Failed to fetch script revision %s: %v", test.LatestRevisionID, err)
		} else {
			// Strip plugin import to return clean user script
			cleanScript, err := scriptprocessor.StripHarnessPluginBase64(revision.ScriptContent)
			if err != nil {
				log.Printf("[GetLoadTest] Failed to strip plugin from script: %v", err)
			} else {
				cleanScriptContent = cleanScript
			}
		}
	}

	response := toLoadTestResponse(test)
	response.ScriptContent = cleanScriptContent

	respondJSON(w, http.StatusOK, response)
}

// ListLoadTests godoc
// @Summary List all load tests
// @Description Returns a list of all load test configurations with optional filtering and sorting
// @Tags LoadTests
// @Produce json
// @Param name query string false "Filter by name (partial match)"
// @Param sortBy query string false "Sort by field: createdAt or updatedAt" default(createdAt)
// @Param sortOrder query string false "Sort order: asc or desc" default(desc)
// @Success 200 {array} LoadTestResponse "List of load tests"
// @Failure 500 {object} ErrorResponse "Failed to list load tests"
// @Router /load-tests [get]
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

	if name := query.Get("name"); name != "" {
		filter.Name = &name
	}

	if tags := query["tags"]; len(tags) > 0 {
		filter.Tags = tags
	}

	// Sorting parameters
	if sortBy := query.Get("sortBy"); sortBy != "" {
		filter.SortBy = sortBy
	} else {
		filter.SortBy = "createdAt" // Default
	}

	if sortOrder := query.Get("sortOrder"); sortOrder != "" {
		filter.SortOrder = sortOrder
	} else {
		filter.SortOrder = "desc" // Default
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

// UpdateLoadTest godoc
// @Summary Update load test configuration
// @Description Updates an existing load test configuration (excluding script)
// @Tags LoadTests
// @Accept json
// @Produce json
// @Param id path string true "Load Test ID"
// @Param request body UpdateLoadTestRequest true "Updated load test configuration"
// @Success 200 {object} LoadTestResponse "Load test updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 404 {object} ErrorResponse "Load test not found"
// @Failure 500 {object} ErrorResponse "Failed to update load test"
// @Router /load-tests/{id} [put]
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

// DeleteLoadTest godoc
// @Summary Delete a load test
// @Description Deletes a load test configuration and all its associated data
// @Tags LoadTests
// @Produce json
// @Param id path string true "Load Test ID"
// @Success 200 {object} SuccessResponse "Load test deleted successfully"
// @Failure 500 {object} ErrorResponse "Failed to delete load test"
// @Router /load-tests/{id} [delete]
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

// CreateLoadTestRun godoc
// @Summary Start a new load test run
// @Description Creates and starts a new load test run using the latest script revision
// @Tags Runs
// @Accept json
// @Produce json
// @Param id path string true "Load Test ID"
// @Param request body CreateLoadTestRunRequest true "Test run configuration"
// @Success 201 {object} LoadTestRunResponse "Load test run started successfully"
// @Failure 400 {object} ErrorResponse "Invalid request or validation error"
// @Failure 404 {object} ErrorResponse "Load test not found or no script available"
// @Failure 500 {object} ErrorResponse "Failed to start load test run"
// @Router /load-tests/{id}/runs [post]
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

	// Get the latest script revision
	latestRevision, err := h.scriptRevisionStore.GetLatestByLoadTestID(loadTestID)
	if err != nil {
		respondError(w, http.StatusNotFound, "No script found for this load test", err)
		return
	}

	nowMillis := time.Now().UnixMilli()
	runID := uuid.New().String()
	run := &domain.LoadTestRun{
		ID:               runID,
		LoadTestID:       loadTestID,
		ScriptRevisionID: latestRevision.ID,
		Name:             req.Name,
		AccountID:        loadTest.AccountID,
		OrgID:            loadTest.OrgID,
		ProjectID:        loadTest.ProjectID,
		EnvID:            loadTest.EnvID,
		TargetUsers:      targetUsers,
		SpawnRate:        spawnRate,
		DurationSeconds:  durationSeconds,
		Status:           domain.LoadTestRunStatusPending,
		CreatedAt:        nowMillis,
		CreatedBy:        req.CreatedBy,
		UpdatedAt:        nowMillis,
		UpdatedBy:        req.CreatedBy,
		Metadata:         req.Metadata,
	}

	if err := h.loadTestRunStore.Create(run); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create load test run", err)
		return
	}

	// Start the actual test via orchestrator
	startReq := &service.CreateTestRunRequest{
		LoadTestRunID:   runID,
		AccountID:       loadTest.AccountID,
		OrgID:           loadTest.OrgID,
		ProjectID:       loadTest.ProjectID,
		EnvID:           loadTest.EnvID,
		ScriptContent:   latestRevision.ScriptContent, // Base64 encoded script
		TargetURL:       loadTest.TargetURL,
		TargetUsers:     targetUsers,
		SpawnRate:       spawnRate,
		DurationSeconds: durationSeconds,
	}

	startedRun, err := h.orchestrator.CreateTestRun(startReq)
	if err != nil {
		// Test creation failed, update run status to Failed
		run.Status = domain.LoadTestRunStatusFailed
		run.UpdatedAt = time.Now().UnixMilli()
		h.loadTestRunStore.Update(run)

		respondError(w, http.StatusInternalServerError, "Failed to start load test", err)
		return
	}

	respondJSON(w, http.StatusCreated, toLoadTestRunResponse(startedRun))
}

// GetLoadTestRun godoc
// @Summary Get load test run details
// @Description Retrieves details and current status of a specific load test run
// @Tags Runs
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Success 200 {object} LoadTestRunResponse "Load test run details"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Router /runs/{id} [get]
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

// ListLoadTestRuns godoc
// @Summary List load test runs
// @Description Returns a list of load test runs, optionally filtered by load test ID or other criteria
// @Tags Runs
// @Produce json
// @Param id path string true "Load Test ID (when using /load-tests/{id}/runs endpoint)"
// @Param accountId query string false "Filter by account ID"
// @Param orgId query string false "Filter by organization ID"
// @Param projectId query string false "Filter by project ID"
// @Param name query string false "Filter by name (partial match)"
// @Param status query string false "Filter by status (Pending, Running, Finished, Failed, Stopped)"
// @Param sortBy query string false "Sort by field: createdAt or updatedAt" default(createdAt)
// @Param sortOrder query string false "Sort order: asc or desc" default(desc)
// @Success 200 {array} LoadTestRunResponse "List of load test runs"
// @Failure 500 {object} ErrorResponse "Failed to list load test runs"
// @Router /runs [get]
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

	if name := query.Get("name"); name != "" {
		filter.Name = &name
	}

	if statusStr := query.Get("status"); statusStr != "" {
		status := domain.LoadTestRunStatus(statusStr)
		filter.Status = &status
	}

	// Sorting parameters
	if sortBy := query.Get("sortBy"); sortBy != "" {
		filter.SortBy = sortBy
	} else {
		filter.SortBy = "createdAt" // Default
	}

	if sortOrder := query.Get("sortOrder"); sortOrder != "" {
		filter.SortOrder = sortOrder
	} else {
		filter.SortOrder = "desc" // Default
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

// StopLoadTestRun godoc
// @Summary Stop a running load test
// @Description Stops a currently running load test run
// @Tags Runs
// @Produce json
// @Param id path string true "Load Test Run ID"
// @Success 200 {object} SuccessResponse "Load test run stopped successfully"
// @Failure 404 {object} ErrorResponse "Load test run not found"
// @Failure 500 {object} ErrorResponse "Failed to stop load test run"
// @Router /runs/{id}/stop [post]
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
