// Package storage provides data persistence for StormDB v2
// Handles test runs, results, and metadata storage
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elchinoo/stormdb/core"
)

// Manager implements the StorageManager interface
type Manager struct {
	db     core.DatabaseManager
	logger core.Logger
}

// NewManager creates a new storage manager
func NewManager(db core.DatabaseManager, logger core.Logger) *Manager {
	return &Manager{
		db:     db,
		logger: logger.WithFields(core.Field{Key: "component", Value: "storage"}),
	}
}

// StoreLog writes a log entry for a test run into the database
func (m *Manager) StoreLog(ctx context.Context, entry core.LogEntry) error {
	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get DB connection for log storage: %w", err)
	}
	_, err = db.ExecContext(ctx,
		`INSERT INTO test_run_logs (test_run_id, timestamp, level, message, metadata) VALUES ($1, $2, $3, $4, $5)`,
		entry.TestRunID, entry.Timestamp, entry.Level, entry.Message, entry.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to insert log entry: %w", err)
	}
	return nil
}

// GetTestRunLogs retrieves logs for a test run
func (m *Manager) GetTestRunLogs(ctx context.Context, testRunID int64, limit int) ([]core.LogEntry, error) {
	query := `
		SELECT test_run_id, timestamp, level, message, metadata
		FROM test_run_logs 
		WHERE test_run_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, testRunID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []core.LogEntry
	for rows.Next() {
		var log core.LogEntry
		err := rows.Scan(&log.TestRunID, &log.Timestamp, &log.Level, &log.Message, &log.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	return logs, nil
}

// FixStuckTests marks any pending or running test runs as failed at Startup
func (m *Manager) FixStuckTests(ctx context.Context) (int64, error) {
	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get DB connection for fixing stuck tests: %w", err)
	}
	result, err := db.ExecContext(ctx,
		`UPDATE test_run SET status = 'failed', end_time = now() WHERE status IN ('pending','running') AND end_time IS NULL`,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to update stuck test runs: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected for stuck tests: %w", err)
	}
	m.logger.Info("Fixed stuck test runs", core.Field{Key: "count", Value: count})
	return count, nil
}

// CreateTestRun creates a new test run record
func (m *Manager) CreateTestRun(ctx context.Context, run *core.TestRun) (int64, error) {
	configJSON, err := json.Marshal(run.Config)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config: %w", err)
	}

	query := `
		INSERT INTO test_run 
		(test_type_id, plugin_id, host, port, db_name, name, description, status, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get database connection: %w", err)
	}

	var id int64
	err = db.QueryRowContext(ctx, query,
		run.TestTypeID,
		run.PluginID,
		run.Host,
		run.Port,
		run.DBName,
		run.Name,
		run.Description,
		run.Status,
		configJSON,
	).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("failed to create test run: %w", err)
	}

	m.logger.Info("test run created",
		core.Field{Key: "test_run_id", Value: id},
		core.Field{Key: "plugin_id", Value: run.PluginID},
		core.Field{Key: "test_type_id", Value: run.TestTypeID},
	)

	return id, nil
}

// UpdateTestRunStatus updates the status of a test run
func (m *Manager) UpdateTestRunStatus(ctx context.Context, runID int64, status core.ServiceStatus) error {
	var query string
	var args []interface{}

	switch status {
	case core.StatusRunning:
		query = "UPDATE test_run SET status = $1, start_time = $2 WHERE id = $3"
		args = []interface{}{status, time.Now(), runID}
	case core.StatusSucceeded, core.StatusFailed, core.StatusAborted:
		query = "UPDATE test_run SET status = $1, end_time = $2 WHERE id = $3"
		args = []interface{}{status, time.Now(), runID}
	default:
		query = "UPDATE test_run SET status = $1 WHERE id = $2"
		args = []interface{}{status, runID}
	}

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update test run status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("test run %d not found", runID)
	}

	m.logger.Info("test run status updated",
		core.Field{Key: "test_run_id", Value: runID},
		core.Field{Key: "status", Value: status},
	)

	return nil
}

// GetTestRun retrieves a test run by ID
func (m *Manager) GetTestRun(ctx context.Context, runID int64) (*core.TestRun, error) {
	query := `
		SELECT id, test_type_id, plugin_id, host, port, db_name, 
			   name, description, status, config, created_at, start_time, end_time
		FROM test_run 
		WHERE id = $1
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var run core.TestRun
	var configJSON []byte
	var startTime, endTime sql.NullTime

	err = db.QueryRowContext(ctx, query, runID).Scan(
		&run.ID,
		&run.TestTypeID,
		&run.PluginID,
		&run.Host,
		&run.Port,
		&run.DBName,
		&run.Name,
		&run.Description,
		&run.Status,
		&configJSON,
		&run.CreatedAt,
		&startTime,
		&endTime,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test run %d not found", runID)
		}
		return nil, fmt.Errorf("failed to get test run: %w", err)
	}

	// Unmarshal config
	if err := json.Unmarshal(configJSON, &run.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle nullable timestamps
	if startTime.Valid {
		run.StartTime = &startTime.Time
	}
	if endTime.Valid {
		run.EndTime = &endTime.Time
	}

	return &run, nil
}

// ListTestRuns retrieves test runs with pagination
func (m *Manager) ListTestRuns(ctx context.Context, limit, offset int) ([]core.TestRun, error) {
	query := `
		SELECT id, test_type_id, plugin_id, host, port, db_name, 
			   name, description, status, config, created_at, start_time, end_time
		FROM test_run 
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list test runs: %w", err)
	}
	defer rows.Close()

	var runs []core.TestRun
	for rows.Next() {
		var run core.TestRun
		var configJSON []byte
		var startTime, endTime sql.NullTime

		err := rows.Scan(
			&run.ID,
			&run.TestTypeID,
			&run.PluginID,
			&run.Host,
			&run.Port,
			&run.DBName,
			&run.Name,
			&run.Description,
			&run.Status,
			&configJSON,
			&run.CreatedAt,
			&startTime,
			&endTime,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan test run: %w", err)
		}

		// Unmarshal config
		if err := json.Unmarshal(configJSON, &run.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		// Handle nullable timestamps
		if startTime.Valid {
			run.StartTime = &startTime.Time
		}
		if endTime.Valid {
			run.EndTime = &endTime.Time
		}

		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// StoreResults stores test results in batch
func (m *Manager) StoreResults(ctx context.Context, results []core.TestResult) error {
	if len(results) == 0 {
		m.logger.Warn("StoreResults called with empty results array")
		return nil
	}

	m.logger.Info("StoreResults called",
		core.Field{Key: "result_count", Value: len(results)},
		core.Field{Key: "first_test_run_id", Value: results[0].TestRunID},
		core.Field{Key: "first_metric_id", Value: results[0].MetricID},
	)

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	// Begin transaction for batch insert
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare insert statement
	query := `
		INSERT INTO test_run_result 
		(test_run_id, test_metric_id, start_time, end_time, value, tags)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Insert each result
	for i, result := range results {
		tagsJSON, err := json.Marshal(result.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags for result %d: %w", i, err)
		}

		m.logger.Debug("Inserting result",
			core.Field{Key: "index", Value: i},
			core.Field{Key: "test_run_id", Value: result.TestRunID},
			core.Field{Key: "metric_id", Value: result.MetricID},
			core.Field{Key: "value", Value: result.Value},
		)

		_, err = stmt.ExecContext(ctx,
			result.TestRunID,
			result.MetricID,
			result.StartTime,
			result.EndTime,
			result.Value,
			tagsJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert result %d: %w", i, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.logger.Info("Successfully stored results to database",
		core.Field{Key: "stored_count", Value: len(results)},
	)

	return nil
}

// GetResults retrieves results for a test run
func (m *Manager) GetResults(ctx context.Context, runID int64) ([]core.TestResult, error) {
	query := `
		SELECT id, test_run_id, test_metric_id, start_time, end_time, value, tags
		FROM test_run_result 
		WHERE test_run_id = $1
		ORDER BY start_time
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get results: %w", err)
	}
	defer rows.Close()

	var results []core.TestResult
	for rows.Next() {
		var result core.TestResult
		var tagsJSON []byte

		err := rows.Scan(
			&result.ID,
			&result.TestRunID,
			&result.MetricID,
			&result.StartTime,
			&result.EndTime,
			&result.Value,
			&tagsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		// Unmarshal tags
		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &result.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// GetResultsByMetric retrieves recent results for a specific metric
func (m *Manager) GetResultsByMetric(ctx context.Context, metricCode string, limit int) ([]core.TestResult, error) {
	query := `
		SELECT trr.id, trr.test_run_id, trr.test_metric_id, trr.start_time, trr.end_time, trr.value, trr.tags
		FROM test_run_result trr
		JOIN test_metric tm ON trr.test_metric_id = tm.id
		WHERE tm.code = $1
		ORDER BY trr.start_time DESC
		LIMIT $2
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, metricCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get results by metric: %w", err)
	}
	defer rows.Close()

	var results []core.TestResult
	for rows.Next() {
		var result core.TestResult
		var tagsJSON []byte

		err := rows.Scan(
			&result.ID,
			&result.TestRunID,
			&result.MetricID,
			&result.StartTime,
			&result.EndTime,
			&result.Value,
			&tagsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan result: %w", err)
		}

		if tagsJSON != nil {
			if err := json.Unmarshal(tagsJSON, &result.Tags); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
			}
		}
		results = append(results, result)
	}

	return results, rows.Err()
}

// RegisterTestType registers a new test type
func (m *Manager) RegisterTestType(ctx context.Context, code, name, description string) (int, error) {
	query := `
		INSERT INTO test_type (code, name, description)
		VALUES ($1, $2, $3)
		ON CONFLICT (code) DO UPDATE SET 
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			updated_at = now()
		RETURNING id
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get database connection: %w", err)
	}

	var id int
	err = db.QueryRowContext(ctx, query, code, name, description).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to register test type: %w", err)
	}

	m.logger.Info("test type registered",
		core.Field{Key: "test_type_id", Value: id},
		core.Field{Key: "code", Value: code},
	)

	return id, nil
}

// GetTestType retrieves a test type by code
func (m *Manager) GetTestType(ctx context.Context, code string) (*core.TestType, error) {
	query := `
		SELECT id, code, name, description, created_at, updated_at, is_active
		FROM test_type 
		WHERE code = $1
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var testType core.TestType
	err = db.QueryRowContext(ctx, query, code).Scan(
		&testType.ID,
		&testType.Code,
		&testType.Name,
		&testType.Description,
		&testType.CreatedAt,
		&testType.UpdatedAt,
		&testType.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("test type %s not found", code)
		}
		return nil, fmt.Errorf("failed to get test type: %w", err)
	}

	return &testType, nil
}

// RegisterPlugin registers a new plugin
func (m *Manager) RegisterPlugin(ctx context.Context, metadata core.PluginMetadata) (int, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO plugin (name, version, sha256, metadata)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, version) DO UPDATE SET 
			sha256 = EXCLUDED.sha256,
			metadata = EXCLUDED.metadata
		RETURNING id
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get database connection: %w", err)
	}

	var id int
	err = db.QueryRowContext(ctx, query, metadata.Name, metadata.Version, metadata.SHA256, metadataJSON).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to register plugin: %w", err)
	}

	m.logger.Info("plugin registered",
		core.Field{Key: "plugin_id", Value: id},
		core.Field{Key: "name", Value: metadata.Name},
		core.Field{Key: "version", Value: metadata.Version},
	)

	return id, nil
}

// GetPlugin retrieves a plugin by name and version
func (m *Manager) GetPlugin(ctx context.Context, name, version string) (*core.PluginMetadata, error) {
	query := `
		SELECT id, metadata
		FROM plugin 
		WHERE name = $1 AND version = $2 AND is_active = true
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var metadataJSON []byte
	var metadata core.PluginMetadata
	err = db.QueryRowContext(ctx, query, name, version).Scan(&metadata.ID, &metadataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("plugin %s v%s not found", name, version)
		}
		return nil, fmt.Errorf("failed to get plugin: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}

// RegisterMetric registers a new test metric
func (m *Manager) RegisterMetric(ctx context.Context, code, description, unit string) (int, error) {
	query := `
		INSERT INTO test_metric (code, description, unit)
		VALUES ($1, $2, $3)
		ON CONFLICT (code) DO UPDATE SET 
			description = EXCLUDED.description,
			unit = EXCLUDED.unit,
			updated_at = now()
		RETURNING id
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get database connection: %w", err)
	}

	var id int
	err = db.QueryRowContext(ctx, query, code, description, unit).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to register metric: %w", err)
	}

	m.logger.Debug("metric registered",
		core.Field{Key: "metric_id", Value: id},
		core.Field{Key: "code", Value: code},
	)

	return id, nil
}

// GetMetric retrieves a metric by code
func (m *Manager) GetMetric(ctx context.Context, code string) (*core.TestMetric, error) {
	query := `
		SELECT id, code, description, unit, created_at, updated_at, is_active
		FROM test_metric 
		WHERE code = $1
	`

	db, err := m.db.GetConnection(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	var metric core.TestMetric
	err = db.QueryRowContext(ctx, query, code).Scan(
		&metric.ID,
		&metric.Code,
		&metric.Description,
		&metric.Unit,
		&metric.CreatedAt,
		&metric.UpdatedAt,
		&metric.IsActive,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("metric %s not found", code)
		}
		return nil, fmt.Errorf("failed to get metric: %w", err)
	}

	return &metric, nil
}
