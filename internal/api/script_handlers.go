package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"Load-manager-cli/internal/domain"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// UpdateScript godoc
// @Summary Update load test script
// @Description Creates a new script revision for the load test with base64 encoded content
// @Tags Scripts
// @Accept json
// @Produce json
// @Param id path string true "Load Test ID"
// @Param request body UpdateScriptRequest true "Script content (base64) and description"
// @Success 200 {object} ScriptRevisionResponse "Script revision created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 404 {object} ErrorResponse "Load test not found"
// @Failure 500 {object} ErrorResponse "Failed to create script revision"
// @Router /load-tests/{id}/script [put]
func (h *Handler) UpdateScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	var req UpdateScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	// Get the load test
	loadTest, err := h.loadTestStore.Get(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Load test not found", err)
		return
	}

	// Get the latest revision to determine the next revision number
	var nextRevisionNumber int
	latestRevision, err := h.scriptRevisionStore.GetLatestByLoadTestID(testID)
	if err != nil {
		// If no revisions exist, start with 1 (shouldn't happen but handle it)
		log.Printf("No previous revisions found for load test %s, starting at 1", testID)
		nextRevisionNumber = 1
	} else {
		nextRevisionNumber = latestRevision.RevisionNumber + 1
	}

	// Create new revision
	nowMillis := time.Now().UnixMilli()
	revisionID := uuid.New().String()
	revision := &domain.ScriptRevision{
		ID:             revisionID,
		LoadTestID:     testID,
		RevisionNumber: nextRevisionNumber,
		ScriptContent:  req.ScriptContent,
		Description:    req.Description,
		CreatedAt:      nowMillis,
		CreatedBy:      req.UpdatedBy,
	}

	if err := h.scriptRevisionStore.Create(revision); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create script revision", err)
		return
	}

	// Update load test to reference the new latest revision
	loadTest.LatestRevisionID = revisionID
	loadTest.UpdatedAt = nowMillis
	loadTest.UpdatedBy = req.UpdatedBy

	if err := h.loadTestStore.Update(loadTest); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update load test", err)
		return
	}

	respondJSON(w, http.StatusOK, toScriptRevisionResponse(revision))
}

// GetScript godoc
// @Summary Get latest script
// @Description Returns the latest script revision for the load test
// @Tags Scripts
// @Produce json
// @Param id path string true "Load Test ID"
// @Success 200 {object} ScriptRevisionResponse "Latest script revision"
// @Failure 404 {object} ErrorResponse "Script not found"
// @Router /load-tests/{id}/script [get]
func (h *Handler) GetScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	// Get the latest revision
	revision, err := h.scriptRevisionStore.GetLatestByLoadTestID(testID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Script not found", err)
		return
	}

	respondJSON(w, http.StatusOK, toScriptRevisionResponse(revision))
}

// GetScriptRevision godoc
// @Summary Get specific script revision
// @Description Returns a specific script revision by its ID
// @Tags Scripts
// @Produce json
// @Param id path string true "Load Test ID"
// @Param revisionId path string true "Revision ID"
// @Success 200 {object} ScriptRevisionResponse "Script revision details"
// @Failure 404 {object} ErrorResponse "Script revision not found"
// @Router /load-tests/{id}/script/revisions/{revisionId} [get]
func (h *Handler) GetScriptRevision(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	revisionID := vars["revisionId"]

	revision, err := h.scriptRevisionStore.Get(revisionID)
	if err != nil {
		respondError(w, http.StatusNotFound, "Script revision not found", err)
		return
	}

	respondJSON(w, http.StatusOK, toScriptRevisionResponse(revision))
}

// ListScriptRevisions godoc
// @Summary List script revision history
// @Description Returns all script revisions for a load test (most recent first)
// @Tags Scripts
// @Produce json
// @Param id path string true "Load Test ID"
// @Param limit query int false "Maximum number of revisions to return" default(10)
// @Success 200 {array} ScriptRevisionResponse "List of script revisions"
// @Failure 500 {object} ErrorResponse "Failed to list script revisions"
// @Router /load-tests/{id}/script/revisions [get]
func (h *Handler) ListScriptRevisions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	// Parse limit parameter (default 10)
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	revisions, err := h.scriptRevisionStore.ListByLoadTestID(testID, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to list script revisions", err)
		return
	}

	responses := make([]*ScriptRevisionResponse, len(revisions))
	for i, revision := range revisions {
		responses[i] = toScriptRevisionResponse(revision)
	}

	respondJSON(w, http.StatusOK, responses)
}
