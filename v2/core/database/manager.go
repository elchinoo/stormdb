// Package database provides database connection management for StormDB v2
// Handles connection pooling, health monitoring, and schema migration
package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/elchinoo/stormdb/v2/core"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Manager implements the DatabaseManager interface
type Manager struct {
	config *core.DatabaseConfig
	db     *sql.DB
	logger core.Logger
}

// NewManager creates a new database manager
func NewManager(config *core.DatabaseConfig, logger core.Logger) *Manager {
	return &Manager{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to the database
func (m *Manager) Connect(ctx context.Context) error {
	connStr := m.buildConnectionString()

	m.logger.Info("connecting to database",
		core.Field{Key: "host", Value: m.config.Host},
		core.Field{Key: "port", Value: m.config.Port},
		core.Field{Key: "database", Value: m.config.Database},
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(m.config.MaxConnections)
	db.SetMaxIdleConns(m.config.MaxIdleConns)

	if m.config.ConnMaxLifetime != "" {
		if duration, err := time.ParseDuration(m.config.ConnMaxLifetime); err == nil {
			db.SetConnMaxLifetime(duration)
		} else {
			m.logger.Warn("invalid connection max lifetime, using default",
				core.Field{Key: "value", Value: m.config.ConnMaxLifetime},
				core.Field{Key: "error", Value: err.Error()},
			)
			db.SetConnMaxLifetime(time.Hour)
		}
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	m.db = db

	m.logger.Info("database connection established",
		core.Field{Key: "max_connections", Value: m.config.MaxConnections},
		core.Field{Key: "max_idle_connections", Value: m.config.MaxIdleConns},
	)

	return nil
}

// GetConnection returns a database connection from the pool
func (m *Manager) GetConnection(ctx context.Context) (*sql.DB, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	// Test connection health
	if err := m.db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database connection unhealthy: %w", err)
	}

	return m.db, nil
}

// GetConnectionPool returns the connection pool directly
func (m *Manager) GetConnectionPool() *sql.DB {
	return m.db
}

// Health checks the database connection health
func (m *Manager) Health(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	// Create a context with timeout for health check
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Test basic connectivity
	if err := m.db.PingContext(healthCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Test a simple query to ensure the database is responsive
	var result int
	query := "SELECT 1"
	if err := m.db.QueryRowContext(healthCtx, query).Scan(&result); err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	if result != 1 {
		return fmt.Errorf("database query returned unexpected result: %d", result)
	}

	return nil
}

// Close closes the database connection
func (m *Manager) Close() error {
	if m.db == nil {
		return nil
	}

	m.logger.Info("closing database connection")

	if err := m.db.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	m.db = nil
	return nil
}

// Migrate applies database schema migrations
func (m *Manager) Migrate(ctx context.Context) error {
	if m.db == nil {
		return fmt.Errorf("database not connected")
	}

	m.logger.Info("starting database migration")

	// Create migrations table if it doesn't exist
	if err := m.createMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedMigrations, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	// Find migration files
	migrationFiles, err := m.findMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}

	// Apply pending migrations
	for _, migrationFile := range migrationFiles {
		migrationName := strings.TrimSuffix(filepath.Base(migrationFile), ".sql")

		// Skip if already applied
		if appliedMigrations[migrationName] {
			m.logger.Debug("migration already applied", core.Field{Key: "migration", Value: migrationName})
			continue
		}

		m.logger.Info("applying migration", core.Field{Key: "migration", Value: migrationName})

		if err := m.applyMigration(ctx, migrationFile, migrationName); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migrationName, err)
		}

		m.logger.Info("migration applied successfully", core.Field{Key: "migration", Value: migrationName})
	}

	m.logger.Info("database migration completed")
	return nil
}

// BeginTransaction starts a new database transaction
func (m *Manager) BeginTransaction(ctx context.Context) (*sql.Tx, error) {
	if m.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return tx, nil
}

// GetStats returns database connection pool statistics
func (m *Manager) GetStats() sql.DBStats {
	if m.db == nil {
		return sql.DBStats{}
	}
	return m.db.Stats()
}

// buildConnectionString constructs the PostgreSQL connection string
func (m *Manager) buildConnectionString() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		m.config.Host,
		m.config.Port,
		m.config.Username,
		m.config.Password,
		m.config.Database,
		m.config.SSLMode,
	)
}

// createMigrationsTable creates the schema_migrations table if it doesn't exist
func (m *Manager) createMigrationsTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_name VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT now()
		)
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// getAppliedMigrations returns a map of applied migration names
func (m *Manager) getAppliedMigrations(ctx context.Context) (map[string]bool, error) {
	query := "SELECT migration_name FROM schema_migrations"
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var migrationName string
		if err := rows.Scan(&migrationName); err != nil {
			return nil, err
		}
		applied[migrationName] = true
	}

	return applied, rows.Err()
}

// findMigrationFiles finds all SQL migration files in the migrations directory
func (m *Manager) findMigrationFiles() ([]string, error) {
	migrationDir := "./migrations"

	// Check if migrations directory exists
	matches, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
	if err != nil {
		return nil, err
	}

	// Sort migration files to ensure they're applied in order
	sort.Strings(matches)

	return matches, nil
}

// applyMigration applies a single migration file
func (m *Manager) applyMigration(ctx context.Context, migrationFile, migrationName string) error {
	// Read migration file
	content, err := readFile(migrationFile)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Begin transaction for migration
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin migration transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// Record migration as applied
	_, err = tx.ExecContext(ctx,
		"INSERT INTO schema_migrations (migration_name) VALUES ($1)",
		migrationName,
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration transaction: %w", err)
	}

	return nil
}

// readFile reads a file from the filesystem (wrapper for testing)
var readFile = func(filename string) ([]byte, error) {
	return fs.ReadFile(nil, filename)
}

// ExecuteInTransaction executes a function within a database transaction
func (m *Manager) ExecuteInTransaction(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := m.BeginTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
