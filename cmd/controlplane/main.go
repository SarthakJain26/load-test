package main

import (
	"Load-manager-cli/internal/api"
	"Load-manager-cli/internal/config"
	"Load-manager-cli/internal/mongodb"
	"Load-manager-cli/internal/service"
	"Load-manager-cli/internal/store"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "Load-manager-cli/docs" // Import generated swagger docs
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Load Manager API
// @version 1.0
// @description Load testing management platform for creating, executing, and monitoring distributed load tests with Locust.
// @description
// @description ## Features
// @description - **Load Test Management**: Create and manage load tests with script versioning
// @description - **Script Revisions**: Full version control for test scripts with audit trail
// @description - **Test Execution**: Start and monitor load test runs with real-time metrics
// @description - **Visualization**: Real-time dashboards and historical metrics
// @description
// @description ## Authentication
// @description Currently, this API does not require authentication. Future versions will include API key or OAuth2 support.
//
// @contact.name API Support
// @contact.email support@loadmanager.io
//
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
//
// @host localhost:8080
// @BasePath /v1
// @schemes http https
//
// @tag.name LoadTests
// @tag.description Load test configuration and management
//
// @tag.name Scripts
// @tag.description Script revision management with version control
//
// @tag.name Runs
// @tag.description Load test run execution and monitoring
//
// @tag.name Visualization
// @tag.description Real-time metrics and historical data visualization

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	log.Printf("Loading configuration from %s", *configPath)
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded successfully")
	log.Printf("Server will listen on %s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("Configured %d Locust cluster(s)", len(cfg.LocustClusters))

	// Initialize MongoDB
	log.Println("Connecting to MongoDB...")
	mongoClient, err := mongodb.NewClient(mongodb.Config{
		URI:            cfg.MongoDB.URI,
		Database:       cfg.MongoDB.Database,
		ConnectTimeout: time.Duration(cfg.MongoDB.ConnectTimeoutSeconds) * time.Second,
		MaxPoolSize:    uint64(cfg.MongoDB.MaxPoolSize),
	})
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := mongoClient.Close(context.Background()); err != nil {
			log.Printf("Error closing MongoDB connection: %v", err)
		}
	}()
	log.Println("Connected to MongoDB successfully")

	// Initialize MongoDB stores
	loadTestStore, err := store.NewMongoLoadTestStore(mongoClient.Database())
	if err != nil {
		log.Fatalf("Failed to initialize load test store: %v", err)
	}
	log.Println("Load test store initialized with indexes")

	loadTestRunStore, err := store.NewMongoLoadTestRunStore(mongoClient.Database())
	if err != nil {
		log.Fatalf("Failed to initialize load test run store: %v", err)
	}
	log.Println("Load test run store initialized with indexes")

	metricsStore, err := store.NewMongoMetricsStore(mongoClient.Database())
	if err != nil {
		log.Fatalf("Failed to initialize metrics store: %v", err)
	}
	log.Println("Metrics time-series store initialized with indexes")

	scriptRevisionStore, err := store.NewMongoScriptRevisionStore(mongoClient.Database())
	if err != nil {
		log.Fatalf("Failed to initialize script revision store: %v", err)
	}
	log.Println("Script revision store initialized with indexes")

	// Initialize orchestrator
	orchestrator := service.NewOrchestrator(cfg, loadTestStore, loadTestRunStore, metricsStore)
	orchestrator.Start()
	log.Println("Orchestrator started")

	// Initialize API handlers
	handler := api.NewHandler(orchestrator, loadTestStore, loadTestRunStore, scriptRevisionStore, cfg)
	visualizationHandler := api.NewVisualizationHandler(loadTestRunStore, metricsStore)

	// Setup router
	router := setupRouter(handler, visualizationHandler)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	log.Println("Control Plane is running")
	log.Printf("API available at http://%s/v1", addr)

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop orchestrator
	orchestrator.Stop()

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// setupRouter configures all API routes
func setupRouter(handler *api.Handler, visualizationHandler *api.VisualizationHandler) *mux.Router {
	router := mux.NewRouter()

	// Apply auth middleware to all routes
	router.Use(handler.AuthMiddleware)

	// Health check (no auth required)
	router.HandleFunc("/health", handler.Health).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/v1").Subrouter()

	// LoadTest management endpoints
	v1.HandleFunc("/load-tests", handler.CreateLoadTest).Methods("POST")
	v1.HandleFunc("/load-tests", handler.ListLoadTests).Methods("GET")
	v1.HandleFunc("/load-tests/{id}", handler.GetLoadTest).Methods("GET")
	v1.HandleFunc("/load-tests/{id}", handler.UpdateLoadTest).Methods("PUT")
	v1.HandleFunc("/load-tests/{id}", handler.DeleteLoadTest).Methods("DELETE")

	// Script management endpoints
	v1.HandleFunc("/load-tests/{id}/script", handler.UpdateScript).Methods("PUT")
	v1.HandleFunc("/load-tests/{id}/script", handler.GetScript).Methods("GET")
	v1.HandleFunc("/load-tests/{id}/script/revisions", handler.ListScriptRevisions).Methods("GET")
	v1.HandleFunc("/load-tests/{id}/script/revisions/{revisionId}", handler.GetScriptRevision).Methods("GET")

	// LoadTestRun execution endpoints
	v1.HandleFunc("/load-tests/{id}/runs", handler.CreateLoadTestRun).Methods("POST")
	v1.HandleFunc("/load-tests/{id}/runs", handler.ListLoadTestRuns).Methods("GET")
	v1.HandleFunc("/runs", handler.ListLoadTestRuns).Methods("GET")
	v1.HandleFunc("/runs/{id}", handler.GetLoadTestRun).Methods("GET")
	v1.HandleFunc("/runs/{id}/stop", handler.StopLoadTestRun).Methods("POST")

	// Optimized visualization endpoints for dashboard UI
	v1.HandleFunc("/runs/{id}/graph", visualizationHandler.GetRunGraph).Methods("GET")
	v1.HandleFunc("/runs/{id}/summary", visualizationHandler.GetRunSummary).Methods("GET")
	v1.HandleFunc("/runs/{id}/requests", visualizationHandler.GetLiveRequestLog).Methods("GET")

	// Detailed visualization endpoints for charts and metrics
	v1.HandleFunc("/runs/{id}/metrics/timeseries", visualizationHandler.GetTimeseriesChart).Methods("GET")
	v1.HandleFunc("/runs/{id}/metrics/scatter", visualizationHandler.GetScatterPlot).Methods("GET")
	v1.HandleFunc("/runs/{id}/metrics/aggregate", visualizationHandler.GetAggregatedStats).Methods("GET")

	// Swagger documentation
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	// Internal Locust callback endpoints
	internal := v1.PathPrefix("/internal/locust").Subrouter()
	internal.HandleFunc("/test-start", handler.LocustCallbackTestStart).Methods("POST")
	internal.HandleFunc("/test-stop", handler.LocustCallbackTestStop).Methods("POST")
	internal.HandleFunc("/metrics", handler.LocustCallbackMetrics).Methods("POST")
	internal.HandleFunc("/register-external", handler.RegisterExternalTest).Methods("POST")

	// Log all registered routes
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			methods, _ := route.GetMethods()
			log.Printf("Route: %v %s", methods, pathTemplate)
		}
		return nil
	})

	return router
}
