package main

import (
	"Load-manager-cli/internal/api"
	"Load-manager-cli/internal/config"
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

	"github.com/gorilla/mux"
)

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

	// Initialize store
	testRunStore := store.NewInMemoryTestRunStore()
	log.Println("In-memory store initialized")

	// Initialize orchestrator
	orchestrator := service.NewOrchestrator(cfg, testRunStore)
	orchestrator.Start()
	log.Println("Orchestrator started")

	// Initialize API handler
	handler := api.NewHandler(orchestrator, cfg)

	// Setup router
	router := setupRouter(handler)

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
func setupRouter(handler *api.Handler) *mux.Router {
	router := mux.NewRouter()

	// Apply auth middleware to all routes
	router.Use(handler.AuthMiddleware)

	// Health check (no auth required)
	router.HandleFunc("/health", handler.Health).Methods("GET")

	// API v1 routes
	v1 := router.PathPrefix("/v1").Subrouter()

	// Test management endpoints
	v1.HandleFunc("/tests", handler.CreateTest).Methods("POST")
	v1.HandleFunc("/tests", handler.ListTests).Methods("GET")
	v1.HandleFunc("/tests/{id}", handler.GetTest).Methods("GET")
	v1.HandleFunc("/tests/{id}/stop", handler.StopTest).Methods("POST")

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

// Delegate 1.0
// Dependencies
// Docker, Cannot run on user's system
// Don't work on VMs (don't run on containerless)

// Delegate 2.0
// Dependencies
// None
