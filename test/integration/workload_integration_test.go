package integration_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/elchinoo/stormdb/internal/database"
	"github.com/elchinoo/stormdb/internal/workload"
	"github.com/elchinoo/stormdb/pkg/types"
)

// TestDatabaseConnection tests basic database connectivity
func TestDatabaseConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)

	ctx := context.Background()
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	// Test basic query
	var result int
	err = db.Pool.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute basic query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

// TestSimpleWorkloadSetup tests that simple workload can set up its schema
func TestSimpleWorkloadSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	cfg.Workload = "simple"

	ctx := context.Background()
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	// Create factory and get workload
	factory, err := workload.NewFactory(cfg)
	if err != nil {
		t.Fatalf("Failed to create workload factory: %v", err)
	}
	defer factory.Cleanup()

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Failed to initialize factory: %v", err)
	}

	w, err := factory.Get("simple")
	if err != nil {
		t.Fatalf("Failed to get simple workload: %v", err)
	}

	// Test setup
	err = w.Setup(ctx, db.Pool, cfg)
	if err != nil {
		t.Fatalf("Failed to setup simple workload: %v", err)
	}

	// Verify table was created
	var tableName string
	err = db.Pool.QueryRow(ctx, "SELECT tablename FROM pg_tables WHERE tablename = 'loadtest'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to verify table creation: %v", err)
	}

	if tableName != "loadtest" {
		t.Errorf("Expected table 'loadtest', got '%s'", tableName)
	}

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS loadtest CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}
}

// TestTPCCWorkloadSetup tests that TPCC workload can set up its schema
func TestTPCCWorkloadSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	cfg.Workload = "tpcc"
	cfg.Scale = 1 // Small scale for testing

	ctx := context.Background()
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	// Create factory and get workload
	factory, err := workload.NewFactory(cfg)
	if err != nil {
		t.Fatalf("Failed to create workload factory: %v", err)
	}
	defer func() { _ = factory.Cleanup() }()

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Failed to initialize factory: %v", err)
	}

	w, err := factory.Get("tpcc")
	if err != nil {
		t.Fatalf("Failed to get tpcc workload: %v", err)
	}

	// Test setup
	err = w.Setup(ctx, db.Pool, cfg)
	if err != nil {
		t.Fatalf("Failed to setup tpcc workload: %v", err)
	}

	// Verify core tables were created
	tables := []string{"warehouse", "district", "customer", "orders", "order_line"}
	for _, table := range tables {
		var tableName string
		err = db.Pool.QueryRow(ctx,
			"SELECT tablename FROM pg_tables WHERE tablename = $1", table).Scan(&tableName)
		if err != nil {
			t.Fatalf("Failed to verify table '%s' creation: %v", table, err)
		}
	}

	// Cleanup
	for _, table := range []string{"order_line", "orders", "customer", "district", "warehouse"} {
		_, err = db.Pool.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: Failed to cleanup table %s: %v", table, err)
		}
	}
}

// TestVectorWorkloadSetup tests that vector workload (if available via plugin) can set up its schema
func TestVectorWorkloadSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg := getTestConfig(t)
	cfg.Workload = "vector_1024"
	// Configure plugin paths
	cfg.Plugins.Paths = []string{"../../build/plugins"}
	cfg.Plugins.AutoLoad = true

	ctx := context.Background()
	db, err := database.NewPostgres(cfg)
	if err != nil {
		t.Skipf("Could not connect to test database: %v", err)
	}
	defer db.Close()

	// Check if pgvector extension is available
	var extName string
	err = db.Pool.QueryRow(ctx, "SELECT extname FROM pg_extension WHERE extname = 'vector'").Scan(&extName)
	if err != nil {
		t.Skip("pgvector extension not available, skipping vector workload test")
	}

	// Create factory and get workload
	factory, err := workload.NewFactory(cfg)
	if err != nil {
		t.Fatalf("Failed to create workload factory: %v", err)
	}
	defer func() { _ = factory.Cleanup() }()

	if err := factory.Initialize(); err != nil {
		t.Fatalf("Failed to initialize factory: %v", err)
	}

	// Discover plugins to load vector plugin if available
	_, err = factory.DiscoverPlugins()
	if err != nil {
		t.Logf("Plugin discovery warning: %v", err)
	}

	w, err := factory.Get("vector_1024")
	if err != nil {
		t.Skipf("Vector workload not available (likely plugin not built): %v", err)
	}

	// Test setup
	err = w.Setup(ctx, db.Pool, cfg)
	if err != nil {
		t.Fatalf("Failed to setup vector_1024 workload: %v", err)
	}

	// Verify table was created
	var tableName string
	err = db.Pool.QueryRow(ctx, "SELECT tablename FROM pg_tables WHERE tablename = 'items_1024'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Failed to verify table creation: %v", err)
	}

	// Cleanup
	_, err = db.Pool.Exec(ctx, "DROP TABLE IF EXISTS items_1024 CASCADE")
	if err != nil {
		t.Logf("Warning: Failed to cleanup table: %v", err)
	}
}

// TestWorkloadExecution tests short execution of each workload
func TestWorkloadExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		workload string
		scale    int
	}{
		{"Simple workload", "simple", 10},
		{"TPCC workload", "tpcc", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := getTestConfig(t)
			cfg.Workload = tc.workload
			cfg.Scale = tc.scale
			cfg.Duration = "2s" // Very short test
			cfg.Workers = 1
			cfg.Connections = 2

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			db, err := database.NewPostgres(cfg)
			if err != nil {
				t.Skipf("Could not connect to test database: %v", err)
			}
			defer db.Close()

			// Create factory and get workload
			factory, err := workload.NewFactory(cfg)
			if err != nil {
				t.Fatalf("Failed to create workload factory: %v", err)
			}
			defer factory.Cleanup()

			if err := factory.Initialize(); err != nil {
				t.Fatalf("Failed to initialize factory: %v", err)
			}

			w, err := factory.Get(tc.workload)
			if err != nil {
				t.Fatalf("Failed to get workload: %v", err)
			}

			// Setup and cleanup
			err = w.Cleanup(ctx, db.Pool, cfg)
			if err != nil {
				t.Fatalf("Failed to cleanup workload: %v", err)
			}

			// Run short test
			metrics := &types.Metrics{
				ErrorTypes: make(map[string]int64),
			}
			metrics.InitializeLatencyHistogram()

			err = w.Run(ctx, db.Pool, cfg, metrics)
			if err != nil {
				t.Fatalf("Failed to run workload: %v", err)
			}

			// Basic validation
			if metrics.TPS == 0 && metrics.Errors == 0 {
				t.Error("Expected some transactions or errors, got neither")
			}
		})
	}
}

// Helper function to get test configuration
func getTestConfig(t *testing.T) *types.Config {
	cfg := &types.Config{
		Database: struct {
			Type     string `mapstructure:"type"`
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Dbname   string `mapstructure:"dbname"`
			Username string `mapstructure:"username"`
			Password string `mapstructure:"password"`
			Sslmode  string `mapstructure:"sslmode"`
		}{
			Type:     "postgres",
			Host:     getEnvOrDefault("STORMDB_TEST_HOST", "localhost"),
			Port:     5432,
			Dbname:   getEnvOrDefault("STORMDB_TEST_DB", "postgres"),
			Username: getEnvOrDefault("STORMDB_TEST_USER", "postgres"),
			Password: getEnvOrDefault("STORMDB_TEST_PASSWORD", ""),
			Sslmode:  "disable",
		},
		Workload:    "simple",
		Scale:       10,
		Duration:    "1s",
		Workers:     1,
		Connections: 2,
	}

	return cfg
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
