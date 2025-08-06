// Package core provides the central coordination for StormDB v2
package core

import (
	"context"
	"fmt"
	"time"
)

// CoreServicesBuilder helps construct the core services bundle
type CoreServicesBuilder struct {
	Database  DatabaseManager
	Logger    Logger
	Storage   StorageManager
	Config    ConfigManager
	Scheduler SchedulerManager
	Plugin    PluginManager
}

// Build creates a CoreServices instance
func (b *CoreServicesBuilder) Build() *CoreServices {
	return &CoreServices{
		Database:  b.Database,
		Logger:    b.Logger,
		Storage:   b.Storage,
		Config:    b.Config,
		Scheduler: b.Scheduler,
		Plugin:    b.Plugin,
	}
}

// ValidateServices validates that all required services are provided
func (b *CoreServicesBuilder) ValidateServices() error {
	if b.Database == nil {
		return fmt.Errorf("database manager is required")
	}
	if b.Logger == nil {
		return fmt.Errorf("logger is required")
	}
	if b.Storage == nil {
		return fmt.Errorf("storage manager is required")
	}
	if b.Config == nil {
		return fmt.Errorf("config manager is required")
	}
	if b.Scheduler == nil {
		return fmt.Errorf("scheduler manager is required")
	}
	if b.Plugin == nil {
		return fmt.Errorf("plugin manager is required")
	}
	return nil
}

// HealthChecker provides health checking functionality
type HealthChecker struct {
	services *CoreServices
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(services *CoreServices) *HealthChecker {
	return &HealthChecker{services: services}
}

// CheckHealth performs a comprehensive health check
func (h *HealthChecker) CheckHealth(ctx context.Context) error {
	// Check database health
	if err := h.services.Database.Health(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check scheduler health
	if !h.services.Scheduler.IsRunning() {
		return fmt.Errorf("scheduler is not running")
	}

	h.services.Logger.Debug("health check passed")
	return nil
}

// GetSystemStatus returns comprehensive system status
func (h *HealthChecker) GetSystemStatus() map[string]interface{} {
	status := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"services":  make(map[string]interface{}),
	}

	// Scheduler status
	schedulerStatus := h.services.Scheduler.GetStatus()
	status["services"].(map[string]interface{})["scheduler"] = schedulerStatus

	// Plugin status
	plugins := h.services.Plugin.ListPlugins()
	status["services"].(map[string]interface{})["plugins"] = map[string]interface{}{
		"loaded_count": len(plugins),
		"plugins":      plugins,
	}

	return status
}

// ServiceInitializer provides service initialization helpers
type ServiceInitializer struct {
	logger Logger
}

// NewServiceInitializer creates a new service initializer
func NewServiceInitializer(logger Logger) *ServiceInitializer {
	return &ServiceInitializer{logger: logger}
}

// InitializePlugins initializes all loaded plugins with core services
func (s *ServiceInitializer) InitializePlugins(pluginManager PluginManager, coreServices *CoreServices) error {
	plugins := pluginManager.GetLoadedPlugins()
	s.logger.Info("initializing plugins", Field{Key: "count", Value: len(plugins)})

	for _, plugin := range plugins {
		metadata := plugin.Metadata()
		s.logger.Info("initializing plugin",
			Field{Key: "name", Value: metadata.Name},
			Field{Key: "version", Value: metadata.Version},
		)

		ctx := context.Background()
		if err := plugin.Initialize(ctx, coreServices); err != nil {
			s.logger.Error("failed to initialize plugin",
				Field{Key: "plugin", Value: metadata.Name},
				Field{Key: "error", Value: err.Error()},
			)
			continue
		}

		s.logger.Info("plugin initialized successfully",
			Field{Key: "name", Value: metadata.Name},
		)
	}

	return nil
}

// ServiceCoordinator coordinates the lifecycle of core services
type ServiceCoordinator struct {
	services *CoreServices
	logger   Logger
}

// NewServiceCoordinator creates a new service coordinator
func NewServiceCoordinator(services *CoreServices) *ServiceCoordinator {
	return &ServiceCoordinator{
		services: services,
		logger:   services.Logger.WithFields(Field{Key: "component", Value: "coordinator"}),
	}
}

// StartServices starts all managed services
func (c *ServiceCoordinator) StartServices() error {
	c.logger.Info("starting core services")

	// Start scheduler
	if err := c.services.Scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Load plugins
	c.logger.Info("loading plugins")
	if err := c.services.Plugin.LoadPlugins(); err != nil {
		c.logger.Warn("plugin loading completed with errors",
			Field{Key: "error", Value: err.Error()},
		)
	}

	// Initialize plugins
	initializer := NewServiceInitializer(c.logger)
	if err := initializer.InitializePlugins(c.services.Plugin, c.services); err != nil {
		c.logger.Warn("plugin initialization completed with errors",
			Field{Key: "error", Value: err.Error()},
		)
	}

	c.logger.Info("core services started successfully")
	return nil
}

// StopServices stops all managed services
func (c *ServiceCoordinator) StopServices() error {
	c.logger.Info("stopping core services")

	// Stop scheduler
	if err := c.services.Scheduler.Stop(); err != nil {
		c.logger.Error("failed to stop scheduler", Field{Key: "error", Value: err.Error()})
	}

	// Unload plugins
	if err := c.services.Plugin.UnloadPlugins(); err != nil {
		c.logger.Error("failed to unload plugins", Field{Key: "error", Value: err.Error()})
	}

	// Close database connection
	if err := c.services.Database.Close(); err != nil {
		c.logger.Error("failed to close database", Field{Key: "error", Value: err.Error()})
	}

	c.logger.Info("core services stopped")
	return nil
}
