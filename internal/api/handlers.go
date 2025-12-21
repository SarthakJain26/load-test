package api

import (
	"Load-manager-cli/internal/config"
	"Load-manager-cli/internal/domain"
	"Load-manager-cli/internal/service"
	"Load-manager-cli/internal/store"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

// Handler contains all HTTP handlers for the API
type Handler struct {
	orchestrator *service.Orchestrator
	config       *config.Config
}

// NewHandler creates a new API handler
func NewHandler(orchestrator *service.Orchestrator, config *config.Config) *Handler {
	return &Handler{
		orchestrator: orchestrator,
		config:       config,
	}
}

// CreateTest handles POST /v1/tests - creates and starts a new load test
func (h *Handler) CreateTest(w http.ResponseWriter, r *http.Request) {
	var req CreateTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	// Validate request
	if req.TenantID == "" || req.EnvID == "" || req.ScenarioID == "" {
		respondError(w, http.StatusBadRequest, "tenantId, envId, and scenarioId are required", nil)
		return
	}
	if req.TargetUsers < 1 {
		respondError(w, http.StatusBadRequest, "targetUsers must be at least 1", nil)
		return
	}
	if req.SpawnRate < 0.1 {
		respondError(w, http.StatusBadRequest, "spawnRate must be at least 0.1", nil)
		return
	}
	
	// Create test run
	run, err := h.orchestrator.CreateTestRun(&service.CreateTestRunRequest{
		TenantID:        req.TenantID,
		EnvID:           req.EnvID,
		ScenarioID:      req.ScenarioID,
		TargetUsers:     req.TargetUsers,
		SpawnRate:       req.SpawnRate,
		DurationSeconds: req.DurationSeconds,
		Metadata:        req.Metadata,
	})
	
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create test run", err)
		return
	}
	
	respondJSON(w, http.StatusCreated, toTestRunResponse(run))
}

// StopTest handles POST /v1/tests/{id}/stop - stops a running test
func (h *Handler) StopTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]
	
	if err := h.orchestrator.StopTestRun(testID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to stop test run", err)
		return
	}
	
	respondJSON(w, http.StatusOK, SuccessResponse{
		Success: true,
		Message: "Test stopped successfully",
	})
}

// GetTest handles GET /v1/tests/{id} - retrieves a test run by ID
func (h *Handler) GetTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]
	
	run, err := h.orchestrator.GetTestRun(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Test run not found", err)
		return
	}
	
	respondJSON(w, http.StatusOK, toTestRunResponse(run))
}

// ListTests handles GET /v1/tests - lists test runs with optional filtering
func (h *Handler) ListTests(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	
	filter := &store.TestRunFilter{}
	
	if tenantID := query.Get("tenantId"); tenantID != "" {
		filter.TenantID = &tenantID
	}
	
	if envID := query.Get("envId"); envID != "" {
		filter.EnvID = &envID
	}
	
	if statusStr := query.Get("status"); statusStr != "" {
		status := domain.TestRunStatus(statusStr)
		filter.Status = &status
	}
	
	runs, err := h.orchestrator.ListTestRuns(filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list test runs", err)
		return
	}
	
	// Convert to response format
	responses := make([]*TestRunResponse, len(runs))
	for i, run := range runs {
		responses[i] = toTestRunResponse(run)
	}
	
	respondJSON(w, http.StatusOK, responses)
}

// LocustCallbackTestStart handles POST /v1/internal/locust/test-start
// Called by Locust when a test starts
func (h *Handler) LocustCallbackTestStart(w http.ResponseWriter, r *http.Request) {
	var req LocustCallbackTestStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	if req.RunID == "" {
		respondError(w, http.StatusBadRequest, "runId is required", nil)
		return
	}
	
	if err := h.orchestrator.HandleTestStart(req.RunID); err != nil {
		log.Printf("Error handling test start callback: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to handle test start", err)
		return
	}
	
	respondJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

// LocustCallbackTestStop handles POST /v1/internal/locust/test-stop
// Called by Locust when a test stops
func (h *Handler) LocustCallbackTestStop(w http.ResponseWriter, r *http.Request) {
	var req LocustCallbackTestStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	if req.RunID == "" {
		respondError(w, http.StatusBadRequest, "runId is required", nil)
		return
	}
	
	finalMetrics := toDomainMetricSnapshot(req.FinalMetrics)
	
	if err := h.orchestrator.HandleTestStop(req.RunID, finalMetrics); err != nil {
		log.Printf("Error handling test stop callback: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to handle test stop", err)
		return
	}
	
	respondJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

// LocustCallbackMetrics handles POST /v1/internal/locust/metrics
// Called by Locust to push periodic metrics
func (h *Handler) LocustCallbackMetrics(w http.ResponseWriter, r *http.Request) {
	var req LocustCallbackMetricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	if req.RunID == "" || req.Metrics == nil {
		respondError(w, http.StatusBadRequest, "runId and metrics are required", nil)
		return
	}
	
	metrics := toDomainMetricSnapshot(req.Metrics)
	
	if err := h.orchestrator.UpdateMetrics(req.RunID, metrics); err != nil {
		log.Printf("Error updating metrics: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to update metrics", err)
		return
	}
	
	respondJSON(w, http.StatusOK, SuccessResponse{Success: true})
}

// RegisterExternalTest handles POST /v1/internal/locust/register-external
// Called by Locust when a test is started from the UI (not via API)
func (h *Handler) RegisterExternalTest(w http.ResponseWriter, r *http.Request) {
	var req RegisterExternalTestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}
	
	if req.TenantID == "" || req.EnvID == "" {
		respondError(w, http.StatusBadRequest, "tenantId and envId are required", nil)
		return
	}
	
	// Default scenario ID if not provided
	if req.ScenarioID == "" {
		req.ScenarioID = "ui-started-test"
	}
	
	orchestratorReq := &service.RegisterExternalTestRunRequest{
		TenantID:    req.TenantID,
		EnvID:       req.EnvID,
		ScenarioID:  req.ScenarioID,
		TargetUsers: req.TargetUsers,
		SpawnRate:   req.SpawnRate,
	}
	
	run, err := h.orchestrator.RegisterExternalTestRun(orchestratorReq)
	if err != nil {
		log.Printf("Error registering external test: %v", err)
		respondError(w, http.StatusInternalServerError, "Failed to register external test", err)
		return
	}
	
	respondJSON(w, http.StatusOK, RegisterExternalTestResponse{
		RunID:   run.ID,
		Message: "External test registered successfully",
	})
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Middleware for API authentication (simple token-based auth)
func (h *Handler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Internal callbacks use different auth
		if strings.HasPrefix(r.URL.Path, "/v1/internal/locust/") {
			h.locustCallbackAuthMiddleware(next).ServeHTTP(w, r)
			return
		}
		
		// Check API token for user-facing endpoints
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			respondError(w, http.StatusUnauthorized, "Missing authorization header", nil)
			return
		}
		
		// Expect "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			respondError(w, http.StatusUnauthorized, "Invalid authorization header format", nil)
			return
		}
		
		token := parts[1]
		if h.config.Security.APIToken != "" && token != h.config.Security.APIToken {
			respondError(w, http.StatusUnauthorized, "Invalid API token", nil)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Middleware for Locust callback authentication
func (h *Handler) locustCallbackAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Locust-Token")
		
		if h.config.Security.LocustCallbackToken != "" && token != h.config.Security.LocustCallbackToken {
			respondError(w, http.StatusUnauthorized, "Invalid Locust callback token", nil)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	resp := ErrorResponse{
		Error:   message,
	}
	if err != nil {
		resp.Message = err.Error()
		log.Printf("API Error: %s - %v", message, err)
	}
	respondJSON(w, status, resp)
}
