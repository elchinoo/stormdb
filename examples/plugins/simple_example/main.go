// Example StormDB Plugin - Simple Counter Workload
// This demonstrates the basic structure of a StormDB plugin
//
// To build and use:
//  1. Build: go build -buildmode=plugin -o simple.so main.go
//  2. Configure stormdb to load this plugin in your YAML config
//  3. Run stormdb as normal
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// These types would normally be imported from stormdb/pkg/plugin and stormdb/pkg/types
// For this example, we define them locally to show the required interface

type PluginMetadata struct {
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Description          string   `json:"description"`
	Author               string   `json:"author"`
	WorkloadTypes        []string `json:"workload_types"`
	RequiredExtensions   []string `json:"required_extensions,omitempty"`
	MinPostgreSQLVersion string   `json:"min_postgresql_version,omitempty"`
	Homepage             string   `json:"homepage,omitempty"`
}

type WorkloadPlugin interface {
	GetMetadata() *PluginMetadata
	CreateWorkload(workloadType string) (Workload, error)
	Initialize() error
	Cleanup() error
}

type Workload interface {
	Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error
	Setup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error
	Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error
}

// Simplified types for the example (these would come from stormdb/pkg/types)
type Config struct {
	Scale   int
	Mode    string
	Workers int
}

type Metrics struct {
	// Simplified metrics interface
}

func (m *Metrics) RecordOperation(duration time.Duration) {
	// Would record the operation time
}

func (m *Metrics) RecordError() {
	// Would record an error
}

// Plugin Implementation
type SimpleExamplePlugin struct{}

func (p *SimpleExamplePlugin) GetMetadata() *PluginMetadata {
	return &PluginMetadata{
		Name:        "simple_example",
		Version:     "1.0.0",
		Description: "Simple counter workload for demonstration purposes",
		Author:      "StormDB Team <stormdb@example.com>",
		WorkloadTypes: []string{
			"simple_counter",
			"simple_counter_read",
			"simple_counter_write",
		},
		RequiredExtensions:   []string{},
		MinPostgreSQLVersion: "12.0",
		Homepage:             "https://github.com/elchinoo/stormdb",
	}
}

func (p *SimpleExamplePlugin) CreateWorkload(workloadType string) (Workload, error) {
	switch workloadType {
	case "simple_counter":
		return &SimpleCounterWorkload{Mode: "mixed"}, nil
	case "simple_counter_read":
		return &SimpleCounterWorkload{Mode: "read"}, nil
	case "simple_counter_write":
		return &SimpleCounterWorkload{Mode: "write"}, nil
	default:
		return nil, fmt.Errorf("unsupported workload type: %s", workloadType)
	}
}

func (p *SimpleExamplePlugin) Initialize() error {
	log.Printf("🔌 Initializing Simple Example Plugin v1.0.0")
	return nil
}

func (p *SimpleExamplePlugin) Cleanup() error {
	log.Printf("🔌 Cleaning up Simple Example Plugin")
	return nil
}

// Workload Implementation
type SimpleCounterWorkload struct {
	Mode string
}

func (w *SimpleCounterWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
	log.Printf("🧹 Cleaning up SimpleCounter schema")

	_, err := db.Exec(ctx, `DROP TABLE IF EXISTS simple_counters CASCADE;`)
	if err != nil {
		return fmt.Errorf("failed to drop table: %w", err)
	}

	log.Printf("✅ SimpleCounter cleanup complete")
	return nil
}

func (w *SimpleCounterWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
	log.Printf("🔧 Setting up SimpleCounter schema")

	// Create the counter table
	_, err := db.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS simple_counters (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100) NOT NULL,
			value BIGINT NOT NULL DEFAULT 0,
			last_updated TIMESTAMP DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_simple_counters_name ON simple_counters(name);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Load initial data
	return w.loadInitialData(ctx, db, cfg)
}

func (w *SimpleCounterWorkload) loadInitialData(ctx context.Context, db *pgxpool.Pool, cfg *Config) error {
	// Check if data already exists
	var count int
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM simple_counters").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check existing data: %w", err)
	}

	if count > 0 {
		log.Printf("📊 SimpleCounter: Using existing data (%d counters)", count)
		return nil
	}

	log.Printf("📊 SimpleCounter: Loading initial data (scale: %d)", cfg.Scale)

	// Create initial counters based on scale factor
	counterCount := cfg.Scale * 10 // 10 counters per scale unit

	for i := 0; i < counterCount; i++ {
		counterName := fmt.Sprintf("counter_%03d", i)
		initialValue := rand.Int63n(1000) // Random initial value 0-999

		_, err := db.Exec(ctx, `
			INSERT INTO simple_counters (name, value) 
			VALUES ($1, $2)
		`, counterName, initialValue)

		if err != nil {
			return fmt.Errorf("failed to insert counter %s: %w", counterName, err)
		}
	}

	log.Printf("✅ SimpleCounter: Loaded %d counters", counterCount)
	return nil
}

func (w *SimpleCounterWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *Config, metrics *Metrics) error {
	// Choose operation based on mode
	switch w.Mode {
	case "read":
		return w.runReadOperation(ctx, db, metrics)
	case "write":
		return w.runWriteOperation(ctx, db, metrics)
	case "mixed":
		// 70% reads, 30% writes
		if rand.Float64() < 0.7 {
			return w.runReadOperation(ctx, db, metrics)
		} else {
			return w.runWriteOperation(ctx, db, metrics)
		}
	default:
		return fmt.Errorf("unsupported workload mode: %s", w.Mode)
	}
}

func (w *SimpleCounterWorkload) runReadOperation(ctx context.Context, db *pgxpool.Pool, metrics *Metrics) error {
	start := time.Now()

	var name string
	var value int64
	var lastUpdated time.Time

	// Read a random counter
	err := db.QueryRow(ctx, `
		SELECT name, value, last_updated 
		FROM simple_counters 
		ORDER BY RANDOM() 
		LIMIT 1
	`).Scan(&name, &value, &lastUpdated)

	duration := time.Since(start)

	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("read operation failed: %w", err)
	}

	metrics.RecordOperation(duration)
	return nil
}

func (w *SimpleCounterWorkload) runWriteOperation(ctx context.Context, db *pgxpool.Pool, metrics *Metrics) error {
	start := time.Now()

	// Increment a random counter
	_, err := db.Exec(ctx, `
		UPDATE simple_counters 
		SET value = value + 1, last_updated = NOW() 
		WHERE name = (
			SELECT name FROM simple_counters 
			ORDER BY RANDOM() 
			LIMIT 1
		)
	`)

	duration := time.Since(start)

	if err != nil {
		metrics.RecordError()
		return fmt.Errorf("write operation failed: %w", err)
	}

	metrics.RecordOperation(duration)
	return nil
}

// Export the plugin symbol - this is required for the plugin system
var ExamplePlugin SimpleExamplePlugin
