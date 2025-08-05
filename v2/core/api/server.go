// Package api provides REST and gRPC API endpoints for StormDB v2
// Handles external communication and control interfaces
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/elchinoo/stormdb/v2/core"
	"github.com/gorilla/mux"
)

// Server implements the API server for StormDB
type Server struct {
	config       *core.APIConfig
	coreServices *core.CoreServices
	logger       core.Logger
	httpServer   *http.Server
	router       *mux.Router
}

// NewServer creates a new API server
func NewServer(config *core.APIConfig, coreServices *core.CoreServices, logger core.Logger) *Server {
	return &Server{
		config:       config,
		coreServices: coreServices,
		logger:       logger.WithFields(core.Field{Key: "component", Value: "api"}),
	}
}

// Start starts the API server
func (s *Server) Start() error {
	s.logger.Info("starting API server",
		core.Field{Key: "host", Value: s.config.Host},
		core.Field{Key: "port", Value: s.config.Port},
	)

	// Initialize router
	s.router = mux.NewRouter()
	s.setupRoutes()

	// Configure HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("HTTP server failed", core.Field{Key: "error", Value: err.Error()})
		}
	}()

	s.logger.Info("API server started", core.Field{Key: "address", Value: addr})
	return nil
}

// Stop stops the API server
func (s *Server) Stop() error {
	if s.httpServer == nil {
		return nil
	}

	s.logger.Info("stopping API server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.logger.Info("API server stopped")
	return nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// System status
	s.router.HandleFunc("/status", s.handleStatus).Methods("GET")

	// Plugin management
	s.router.HandleFunc("/plugins", s.handleListPlugins).Methods("GET")
	s.router.HandleFunc("/plugins/{name}", s.handleGetPlugin).Methods("GET")
	s.router.HandleFunc("/plugins/reload", s.handleReloadPlugins).Methods("POST")

	// Test run management
	s.router.HandleFunc("/test-runs", s.handleCreateTestRun).Methods("POST")
	s.router.HandleFunc("/test-runs", s.handleListTestRuns).Methods("GET")
	s.router.HandleFunc("/test-runs/{id}", s.handleGetTestRun).Methods("GET")
	s.router.HandleFunc("/test-runs/{id}/cancel", s.handleCancelTestRun).Methods("POST")

	// Test results
	s.router.HandleFunc("/test-runs/{id}/results", s.handleGetTestResults).Methods("GET")
	s.router.HandleFunc("/metrics/{code}/results", s.handleGetMetricResults).Methods("GET")

	// Configuration
	s.router.HandleFunc("/config", s.handleGetConfig).Methods("GET")

	// Middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check database health
	if err := s.coreServices.Database.Health(ctx); err != nil {
		s.writeErrorResponse(w, http.StatusServiceUnavailable, "database unhealthy", err)
		return
	}

	// Check scheduler status
	schedulerStatus := s.coreServices.Scheduler.GetStatus()

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"services": map[string]interface{}{
			"database":  "healthy",
			"scheduler": schedulerStatus,
		},
	}

	s.writeJSONResponse(w, http.StatusOK, health)
}

// System status endpoint
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	plugins := s.coreServices.Plugin.ListPlugins()
	schedulerStatus := s.coreServices.Scheduler.GetStatus()

	status := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"plugins": map[string]interface{}{
			"loaded_count": len(plugins),
			"plugins":      plugins,
		},
		"scheduler": schedulerStatus,
	}

	s.writeJSONResponse(w, http.StatusOK, status)
}

// List plugins endpoint
func (s *Server) handleListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := s.coreServices.Plugin.ListPlugins()
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"plugins": plugins,
		"count":   len(plugins),
	})
}

// Get plugin endpoint
func (s *Server) handleGetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	plugin, err := s.coreServices.Plugin.GetPlugin(name)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "plugin not found", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, plugin.Metadata())
}

// Reload plugins endpoint
func (s *Server) handleReloadPlugins(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("reloading plugins via API")

	// Unload existing plugins
	if err := s.coreServices.Plugin.UnloadPlugins(); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to unload plugins", err)
		return
	}

	// Load plugins
	if err := s.coreServices.Plugin.LoadPlugins(); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to load plugins", err)
		return
	}

	plugins := s.coreServices.Plugin.ListPlugins()
	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "plugins reloaded successfully",
		"count":   len(plugins),
	})
}

// Create test run endpoint
func (s *Server) handleCreateTestRun(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PluginName  string                 `json:"plugin_name"`
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		Config      map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid request body", err)
		return
	}

	// Validate required fields
	if req.PluginName == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "plugin_name is required", nil)
		return
	}

	// Schedule test
	ctx := r.Context()
	runID, err := s.coreServices.Scheduler.ScheduleTest(ctx, req.PluginName, req.Config)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to schedule test", err)
		return
	}

	s.writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"test_run_id": runID,
		"status":      "scheduled",
	})
}

// List test runs endpoint
func (s *Server) handleListTestRuns(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0 // default
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	ctx := r.Context()
	runs, err := s.coreServices.Storage.ListTestRuns(ctx, limit, offset)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to list test runs", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"test_runs": runs,
		"count":     len(runs),
		"limit":     limit,
		"offset":    offset,
	})
}

// Get test run endpoint
func (s *Server) handleGetTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid test run ID", err)
		return
	}

	ctx := r.Context()
	testRun, err := s.coreServices.Storage.GetTestRun(ctx, id)
	if err != nil {
		s.writeErrorResponse(w, http.StatusNotFound, "test run not found", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, testRun)
}

// Cancel test run endpoint
func (s *Server) handleCancelTestRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid test run ID", err)
		return
	}

	ctx := r.Context()
	if err := s.coreServices.Scheduler.CancelTest(ctx, id); err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to cancel test", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "test run cancelled",
		"id":      id,
	})
}

// Get test results endpoint
func (s *Server) handleGetTestResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeErrorResponse(w, http.StatusBadRequest, "invalid test run ID", err)
		return
	}

	ctx := r.Context()
	results, err := s.coreServices.Storage.GetResults(ctx, id)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to get results", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"test_run_id": id,
		"results":     results,
		"count":       len(results),
	})
}

// Get metric results endpoint
func (s *Server) handleGetMetricResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	code := vars["code"]

	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	ctx := r.Context()
	results, err := s.coreServices.Storage.GetResultsByMetric(ctx, code, limit)
	if err != nil {
		s.writeErrorResponse(w, http.StatusInternalServerError, "failed to get metric results", err)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"metric_code": code,
		"results":     results,
		"count":       len(results),
		"limit":       limit,
	})
}

// Get configuration endpoint
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	config := s.coreServices.Config.GetGlobal()

	// Sanitize sensitive data
	sanitized := *config
	sanitized.Database.Password = "[REDACTED]"

	s.writeJSONResponse(w, http.StatusOK, sanitized)
}

// Middleware functions

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap ResponseWriter to capture status code
		wrapped := &responseWrapper{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		s.logger.Info("HTTP request",
			core.Field{Key: "method", Value: r.Method},
			core.Field{Key: "path", Value: r.URL.Path},
			core.Field{Key: "status", Value: wrapped.statusCode},
			core.Field{Key: "duration_ms", Value: duration.Milliseconds()},
			core.Field{Key: "remote_addr", Value: r.RemoteAddr},
			core.Field{Key: "user_agent", Value: r.UserAgent()},
		)
	})
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper functions

// writeJSONResponse writes a JSON response
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("failed to encode JSON response", core.Field{Key: "error", Value: err.Error()})
	}
}

// writeErrorResponse writes an error response
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	response := map[string]interface{}{
		"error":     message,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response["details"] = err.Error()
		s.logger.Error("API error",
			core.Field{Key: "message", Value: message},
			core.Field{Key: "error", Value: err.Error()},
			core.Field{Key: "status_code", Value: statusCode},
		)
	}

	s.writeJSONResponse(w, statusCode, response)
}

// responseWrapper wraps http.ResponseWriter to capture status code
type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
