# Route Registration for New Visualization APIs

## New API Routes to Add

Add these routes to your router in `cmd/controlplane/main.go`:

```go
// In your router setup function, add these routes:

// New optimized visualization APIs
router.HandleFunc("/v1/runs/{id}/graph", vizHandler.GetRunGraph).Methods("GET")
router.HandleFunc("/v1/runs/{id}/summary", vizHandler.GetRunSummary).Methods("GET")
router.HandleFunc("/v1/runs/{id}/requests", vizHandler.GetLiveRequestLog).Methods("GET")

// Existing visualization APIs (keep these for detailed analysis)
router.HandleFunc("/v1/runs/{id}/chart", vizHandler.GetTimeseriesChart).Methods("GET")
router.HandleFunc("/v1/runs/{id}/scatter", vizHandler.GetScatterPlot).Methods("GET")
router.HandleFunc("/v1/runs/{id}/stats", vizHandler.GetAggregatedStats).Methods("GET")
```

## Complete Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    
    "Load-manager-cli/internal/api"
    "Load-manager-cli/internal/config"
    "Load-manager-cli/internal/service"
    "Load-manager-cli/internal/store"
    
    "github.com/gorilla/mux"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize stores
    mongoLoadTestStore, err := store.NewMongoLoadTestStore(context.Background(), cfg.MongoDB.URI, cfg.MongoDB.Database)
    if err != nil {
        log.Fatalf("Failed to create MongoDB load test store: %v", err)
    }

    mongoLoadTestRunStore, err := store.NewMongoLoadTestRunStore(context.Background(), cfg.MongoDB.URI, cfg.MongoDB.Database)
    if err != nil {
        log.Fatalf("Failed to create MongoDB load test run store: %v", err)
    }

    mongoMetricsStore, err := store.NewMongoMetricsStore(context.Background(), cfg.MongoDB.URI, cfg.MongoDB.Database)
    if err != nil {
        log.Fatalf("Failed to create MongoDB metrics store: %v", err)
    }

    // Initialize orchestrator
    orchestrator := service.NewOrchestrator(
        cfg,
        mongoLoadTestStore,
        mongoLoadTestRunStore,
        mongoMetricsStore,
    )
    orchestrator.Start()
    defer orchestrator.Stop()

    // Initialize handlers
    handler := api.NewHandler(
        orchestrator,
        mongoLoadTestStore,
        mongoLoadTestRunStore,
    )

    vizHandler := api.NewVisualizationHandler(
        mongoLoadTestRunStore,
        mongoMetricsStore,
    )

    // Setup router
    router := mux.NewRouter()

    // Load Test CRUD operations
    router.HandleFunc("/v1/load-tests", handler.CreateLoadTest).Methods("POST")
    router.HandleFunc("/v1/load-tests/{id}", handler.GetLoadTest).Methods("GET")
    router.HandleFunc("/v1/load-tests", handler.ListLoadTests).Methods("GET")
    router.HandleFunc("/v1/load-tests/{id}", handler.UpdateLoadTest).Methods("PUT")
    router.HandleFunc("/v1/load-tests/{id}", handler.DeleteLoadTest).Methods("DELETE")

    // Load Test Run operations
    router.HandleFunc("/v1/load-tests/{id}/runs", handler.CreateLoadTestRun).Methods("POST")
    router.HandleFunc("/v1/runs/{id}", handler.GetLoadTestRun).Methods("GET")
    router.HandleFunc("/v1/runs", handler.ListLoadTestRuns).Methods("GET")
    router.HandleFunc("/v1/load-tests/{id}/runs", handler.ListLoadTestRuns).Methods("GET")
    router.HandleFunc("/v1/runs/{id}/stop", handler.StopLoadTestRun).Methods("POST")

    // ⭐ NEW: Optimized Visualization APIs for Dashboard
    router.HandleFunc("/v1/runs/{id}/graph", vizHandler.GetRunGraph).Methods("GET")
    router.HandleFunc("/v1/runs/{id}/summary", vizHandler.GetRunSummary).Methods("GET")
    router.HandleFunc("/v1/runs/{id}/requests", vizHandler.GetLiveRequestLog).Methods("GET")

    // Existing detailed visualization APIs
    router.HandleFunc("/v1/runs/{id}/chart", vizHandler.GetTimeseriesChart).Methods("GET")
    router.HandleFunc("/v1/runs/{id}/scatter", vizHandler.GetScatterPlot).Methods("GET")
    router.HandleFunc("/v1/runs/{id}/stats", vizHandler.GetAggregatedStats).Methods("GET")

    // Internal/Callback endpoints (called by Locust)
    router.HandleFunc("/v1/internal/locust/register-external", handler.RegisterExternalTest).Methods("POST")
    router.HandleFunc("/v1/internal/locust/callback/test-start", handler.HandleTestStart).Methods("POST")
    router.HandleFunc("/v1/internal/locust/callback/test-stop", handler.HandleTestStop).Methods("POST")
    router.HandleFunc("/v1/internal/locust/callback/metrics", handler.HandleMetrics).Methods("POST")

    // Health check
    router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    }).Methods("GET")

    // Start server
    log.Printf("Starting Load Manager Control Plane on :8080")
    if err := http.ListenAndServe(":8080", router); err != nil {
        log.Fatalf("Server failed: %v", err)
    }
}
```

## API Endpoints Summary

### For Dashboard UI (New - Optimized)
- `GET /v1/runs/{id}/graph` - Graph data (Users, RPS, Errors over time)
- `GET /v1/runs/{id}/summary` - 4 key metrics cards
- `GET /v1/runs/{id}/requests?limit=50` - Recent requests log

### For Detailed Analysis (Existing)
- `GET /v1/runs/{id}/chart` - Complete timeseries with all percentiles
- `GET /v1/runs/{id}/scatter` - Response time scatter plot
- `GET /v1/runs/{id}/stats` - Comprehensive aggregated statistics

### Load Test Management
- `POST /v1/load-tests` - Create a new load test
- `GET /v1/load-tests` - List all load tests
- `GET /v1/load-tests/{id}` - Get load test details
- `PUT /v1/load-tests/{id}` - Update load test
- `DELETE /v1/load-tests/{id}` - Delete load test

### Test Run Management
- `POST /v1/load-tests/{id}/runs` - Start a new test run
- `GET /v1/runs` - List all runs
- `GET /v1/runs/{id}` - Get run details
- `POST /v1/runs/{id}/stop` - Stop a running test

### Internal/Callbacks (from Locust)
- `POST /v1/internal/locust/register-external` - Register externally started test
- `POST /v1/internal/locust/callback/test-start` - Test start notification
- `POST /v1/internal/locust/callback/test-stop` - Test stop notification
- `POST /v1/internal/locust/callback/metrics` - Periodic metrics update
