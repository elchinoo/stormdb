package schema

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MigrationRecord tracks applied migrations
type MigrationRecord struct {
	Version   string    `json:"version"`
	Name      string    `json:"name"`
	AppliedAt time.Time `json:"applied_at"`
	Checksum  string    `json:"checksum"`
}

// Migration represents a single migration file
type Migration struct {
	Version   string
	Name      string
	Path      string
	Content   string
	Checksum  string
	Direction string // "up" or "down"
}

// MigrationManager handles database schema migrations with tracking
type MigrationManager struct {
	db     *sql.DB
	logger Logger
}

// Logger interface for migration logging
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(db *sql.DB, logger Logger) *MigrationManager {
	return &MigrationManager{
		db:     db,
		logger: logger,
	}
}

// Migrate applies all pending migrations in the specified directory
func (m *MigrationManager) Migrate(ctx context.Context, migrationsDir string) error {
	// Ensure migration tracking table exists
	if err := m.createMigrationTable(ctx); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// Load migration files
	migrations, err := m.loadMigrations(migrationsDir, "up")
	if err != nil {
		return fmt.Errorf("failed to load migrations: %w", err)
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedMap := make(map[string]MigrationRecord)
	for _, rec := range applied {
		appliedMap[rec.Version] = rec
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if _, exists := appliedMap[migration.Version]; exists {
			m.logger.Info("skipping already applied migration", "version", migration.Version, "name", migration.Name)
			continue
		}

		m.logger.Info("applying migration", "version", migration.Version, "name", migration.Name)

		if err := m.applyMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}

		m.logger.Info("migration applied successfully", "version", migration.Version)
	}

	return nil
}

// createMigrationTable creates the schema_migrations table if it doesn't exist
func (m *MigrationManager) createMigrationTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			checksum VARCHAR(64) NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_schema_migrations_applied_at ON schema_migrations(applied_at);
	`
	_, err := m.db.ExecContext(ctx, query)
	return err
}

// loadMigrations loads all migration files from the specified directory
func (m *MigrationManager) loadMigrations(dir string, direction string) ([]Migration, error) {
	pattern := filepath.Join(dir, fmt.Sprintf("*.%s.sql", direction))
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	var migrations []Migration
	for _, file := range files {
		migration, err := m.parseMigrationFile(file, direction)
		if err != nil {
			m.logger.Warn("failed to parse migration file", "file", file, "error", err)
			continue
		}
		migrations = append(migrations, migration)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file and extracts metadata
func (m *MigrationManager) parseMigrationFile(filePath string, direction string) (Migration, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return Migration{}, err
	}

	filename := filepath.Base(filePath)
	// Expected format: 001_migration_name.up.sql
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration filename format: %s", filename)
	}

	version := parts[0]
	nameParts := strings.Split(strings.Join(parts[1:], "_"), fmt.Sprintf(".%s.sql", direction))
	name := nameParts[0]

	checksum := m.calculateChecksum(string(content))

	return Migration{
		Version:   version,
		Name:      name,
		Path:      filePath,
		Content:   string(content),
		Checksum:  checksum,
		Direction: direction,
	}, nil
}

// getAppliedMigrations retrieves all applied migrations from the database
func (m *MigrationManager) getAppliedMigrations(ctx context.Context) ([]MigrationRecord, error) {
	query := `SELECT version, name, applied_at, checksum FROM schema_migrations ORDER BY version`
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		if err := rows.Scan(&record.Version, &record.Name, &record.AppliedAt, &record.Checksum); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// applyMigration applies a single migration within a transaction
func (m *MigrationManager) applyMigration(ctx context.Context, migration Migration) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute the migration
	if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
		return fmt.Errorf("migration execution failed: %w", err)
	}

	// Record the migration
	_, err = tx.ExecContext(ctx,
		`INSERT INTO schema_migrations (version, name, checksum) VALUES ($1, $2, $3)`,
		migration.Version, migration.Name, migration.Checksum)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	return tx.Commit()
}

// calculateChecksum calculates MD5 checksum of migration content
func (m *MigrationManager) calculateChecksum(content string) string {
	// Simple checksum implementation - in production use crypto/md5
	hash := 0
	for _, b := range []byte(content) {
		hash = hash*31 + int(b)
	}
	return fmt.Sprintf("%x", hash)
}

// RollbackToVersion rolls back to a specific migration version
func (m *MigrationManager) RollbackToVersion(ctx context.Context, migrationsDir string, targetVersion string) error {
	// Get applied migrations after target version
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	// Load down migrations
	downMigrations, err := m.loadMigrations(migrationsDir, "down")
	if err != nil {
		return err
	}

	downMap := make(map[string]Migration)
	for _, migration := range downMigrations {
		downMap[migration.Version] = migration
	}

	// Apply rollbacks in reverse order
	for i := len(applied) - 1; i >= 0; i-- {
		record := applied[i]
		if record.Version <= targetVersion {
			break
		}

		downMigration, exists := downMap[record.Version]
		if !exists {
			return fmt.Errorf("no down migration found for version %s", record.Version)
		}

		m.logger.Info("rolling back migration", "version", record.Version, "name", record.Name)

		if err := m.rollbackMigration(ctx, downMigration, record.Version); err != nil {
			return fmt.Errorf("failed to rollback migration %s: %w", record.Version, err)
		}
	}

	return nil
}

// rollbackMigration applies a down migration and removes the record
func (m *MigrationManager) rollbackMigration(ctx context.Context, migration Migration, version string) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Execute the down migration
	if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
		return fmt.Errorf("rollback execution failed: %w", err)
	}

	// Remove the migration record
	_, err = tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	return tx.Commit()
}

// GetMigrationStatus returns the current migration status
func (m *MigrationManager) GetMigrationStatus(ctx context.Context, migrationsDir string) ([]MigrationStatus, error) {
	// Load all available migrations
	availableMigrations, err := m.loadMigrations(migrationsDir, "up")
	if err != nil {
		return nil, err
	}

	// Get applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	appliedMap := make(map[string]MigrationRecord)
	for _, rec := range applied {
		appliedMap[rec.Version] = rec
	}

	var status []MigrationStatus
	for _, migration := range availableMigrations {
		s := MigrationStatus{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: false,
		}

		if record, exists := appliedMap[migration.Version]; exists {
			s.Applied = true
			s.AppliedAt = &record.AppliedAt
		}

		status = append(status, s)
	}

	return status, nil
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version   string     `json:"version"`
	Name      string     `json:"name"`
	Applied   bool       `json:"applied"`
	AppliedAt *time.Time `json:"applied_at,omitempty"`
}

// MigrateEnhanced provides enhanced migration functionality with tracking
func MigrateEnhanced(ctx context.Context, db *sql.DB, migrationsDir string, logger Logger) error {
	manager := NewMigrationManager(db, logger)
	return manager.Migrate(ctx, migrationsDir)
}

type simpleLogger struct{}

func (l *simpleLogger) Info(msg string, fields ...interface{})  {}
func (l *simpleLogger) Error(msg string, fields ...interface{}) {}
func (l *simpleLogger) Warn(msg string, fields ...interface{})  {}
