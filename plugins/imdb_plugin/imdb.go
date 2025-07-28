// internal/workload/imdb/imdb.go
package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elchinoo/stormdb/internal/util"
	"github.com/elchinoo/stormdb/pkg/types"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IMDBWorkload implements realistic IMDB-style database operations
type IMDBWorkload struct {
	Mode string // "read", "write", or "mixed"
}

// minLen returns the minimum of two integers
func minLen(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Setup creates the IMDB schema and optionally loads sample data
func (w *IMDBWorkload) Setup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	// Override mode from config if specified
	if cfg.Mode != "" {
		w.Mode = cfg.Mode
	}

	log.Printf("🎬 Setting up Real IMDB workload schema (mode: %s)...", w.Mode)

	// Check if schema already exists by looking for main tables
	var tableCount int
	err := db.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.tables 
		WHERE table_schema = 'public' 
		AND table_name IN ('movies_normalized_meta', 'movies_normalized_actors', 'movies_viewed_logs')
	`).Scan(&tableCount)
	if err != nil {
		return fmt.Errorf("failed to check existing schema: %w", err)
	}

	if tableCount == 3 {
		log.Printf("✅ Real IMDB schema already exists")
	} else {
		// Load the full IMDB schema from imdb.sql
		log.Printf("📊 Creating Real IMDB schema tables...")

		schemas := []string{
			// movies_normalized_meta - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS movies_normalized_meta (
				ai_myid SERIAL PRIMARY KEY,
				imdb_id VARCHAR(32),
				title VARCHAR(255),
				imdb_rating NUMERIC(5,2),
				year INTEGER,
				country VARCHAR(100),
				overview TEXT,
				json_column JSONB,
				upvotes INTEGER,
				downvotes INTEGER
			)`,

			// movies_normalized_actors - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS movies_normalized_actors (
				ai_actor_id SERIAL PRIMARY KEY,
				actor_id VARCHAR(50),
				actor_name VARCHAR(500)
			)`,

			// movies_normalized_cast - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS movies_normalized_cast (
				inc_id SERIAL PRIMARY KEY,
				ai_actor_id INTEGER,
				ai_myid INTEGER,
				actor_character VARCHAR(500)
			)`,

			// movies_normalized_user_comments - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS movies_normalized_user_comments (
				comment_id SERIAL PRIMARY KEY,
				ai_myid INTEGER,
				rating INTEGER,
				comment TEXT,
				imdb_id VARCHAR(20),
				comment_add_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				comment_update_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
			)`,

			// movies_viewed_logs - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS movies_viewed_logs (
				view_id SERIAL PRIMARY KEY,
				ai_myid INTEGER,
				imdb_id VARCHAR(32),
				watched_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
				watched_user_id INTEGER,
				time_watched_sec INTEGER,
				encoded_data VARCHAR(500),
				json_payload JSONB,
				json_imdb_id VARCHAR(255) GENERATED ALWAYS AS ((json_payload ->> 'imdb_id'::text)) STORED
			)`,

			// voting_count_history - exact structure from imdb.sql
			`CREATE TABLE IF NOT EXISTS voting_count_history (
				ai_myid INTEGER NOT NULL,
				store_time TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
				title VARCHAR(255) NOT NULL,
				imdb_id VARCHAR(20),
				comment_count BIGINT DEFAULT 0 NOT NULL,
				max_rate INTEGER,
				avg_rate NUMERIC(14,4) DEFAULT NULL,
				upvotes NUMERIC(32,0) DEFAULT NULL,
				downvotes NUMERIC(32,0) DEFAULT NULL,
				PRIMARY KEY (title, ai_myid, store_time)
			)`,
		}

		for _, schema := range schemas {
			if _, err := db.Exec(ctx, schema); err != nil {
				return fmt.Errorf("failed to create schema: %w", err)
			}
		}

		log.Printf("✅ Real IMDB schema created successfully")
	}

	// Load sample data if tables are empty
	var movieCount int64
	err = db.QueryRow(ctx, "SELECT COUNT(*) FROM movies_normalized_meta").Scan(&movieCount)
	if err != nil {
		return fmt.Errorf("failed to count movies: %w", err)
	}

	if movieCount == 0 {
		// Determine data loading mode
		dataMode := cfg.DataLoading.Mode
		if dataMode == "" {
			dataMode = "generate" // Default to generate mode
		}

		switch dataMode {
		case "generate":
			log.Printf("📊 Generating sample Real IMDB data...")
			if err := w.loadSampleData(ctx, db, cfg.Scale); err != nil {
				return fmt.Errorf("failed to load sample data: %w", err)
			}
		case "dump":
			log.Printf("📦 Loading data from dump file: %s", cfg.DataLoading.FilePath)
			if err := w.loadFromDump(ctx, db, cfg); err != nil {
				return fmt.Errorf("failed to load data from dump: %w", err)
			}
		case "sql":
			log.Printf("📜 Loading data from SQL file: %s", cfg.DataLoading.FilePath)
			if err := w.loadFromSQL(ctx, db, cfg); err != nil {
				return fmt.Errorf("failed to load data from SQL: %w", err)
			}
		default:
			return fmt.Errorf("unsupported data loading mode: %s", dataMode)
		}
	} else {
		log.Printf("✅ IMDB data already exists (%d movies)", movieCount)
	}

	return nil
}

// Cleanup drops all IMDB tables
func (w *IMDBWorkload) Cleanup(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	log.Printf("🧹 Cleaning up Real IMDB workload...")

	tables := []string{
		"voting_count_history",
		"movies_viewed_logs",
		"movies_normalized_user_comments",
		"movies_normalized_cast",
		"movies_normalized_actors",
		"movies_normalized_meta",
	}
	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	log.Printf("✅ Real IMDB cleanup complete")
	return nil
}

// Run executes the IMDB workload based on the configured mode
func (w *IMDBWorkload) Run(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics) error {
	log.Printf("🎬 Starting IMDB %s workload...", w.Mode)

	var wg sync.WaitGroup
	start := time.Now()

	// Start real-time reporting
	stopReporting := w.startRealTimeReporter(ctx, cfg, metrics, start)
	defer stopReporting()

	// Launch workers
	for i := 0; i < cfg.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			w.worker(ctx, db, cfg, metrics, workerID)
		}(i)
	}

	wg.Wait()
	stopReporting()
	return nil
}

// worker executes database operations based on the workload mode
func (w *IMDBWorkload) worker(ctx context.Context, db *pgxpool.Pool, cfg *types.Config, metrics *types.Metrics, workerID int) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

	for {
		select {
		case <-ctx.Done():
			return
		default:
			start := time.Now()
			var err error
			var operation string

			// Determine operation based on mode
			switch w.Mode {
			case "read":
				operation, err = w.executeReadOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsRead, 1)
			case "write":
				operation, err = w.executeWriteOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsModified, 1)
			case "mixed":
				if rng.Intn(100) < 70 { // 70% reads, 30% writes
					operation, err = w.executeReadOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsRead, 1)
				} else {
					operation, err = w.executeWriteOperation(ctx, db, rng)
					atomic.AddInt64(&metrics.RowsModified, 1)
				}
			default:
				operation, err = w.executeReadOperation(ctx, db, rng)
				atomic.AddInt64(&metrics.RowsRead, 1)
			}

			elapsed := time.Since(start).Nanoseconds()

			// Record metrics
			metrics.Mu.Lock()
			metrics.TransactionDur = append(metrics.TransactionDur, elapsed)
			metrics.Mu.Unlock()

			metrics.RecordLatency(elapsed)

			if err != nil {
				atomic.AddInt64(&metrics.Errors, 1)
				metrics.Mu.Lock()
				metrics.ErrorTypes[fmt.Sprintf("%s: %s", operation, err.Error())]++
				metrics.Mu.Unlock()
			} else {
				atomic.AddInt64(&metrics.TPS, 1)
				atomic.AddInt64(&metrics.QPS, 1)
			}

			// Small think time to simulate realistic usage
			time.Sleep(time.Duration(rng.Intn(50)) * time.Millisecond)
		}
	}
}

// executeReadOperation performs various read-heavy operations
func (w *IMDBWorkload) executeReadOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) (string, error) {
	// Mix of simple index tests and complex analytical queries
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		// Basic index utilization tests (70% of queries)
		w.testIndexedTitleSearch,   // Uses B-tree index on title
		w.testIndexedRatingSearch,  // Uses B-tree index on imdb_rating
		w.testCompositeIndexSearch, // Uses composite index on (country, year, imdb_rating)
		w.testUniqueIndexSearch,    // Uses unique index on imdb_id
		w.testGINIndexSearch,       // Uses GIN index on json_column
		w.testFullTableScan,        // Forces full table scan (no index on overview)
		w.testSimpleJoin,           // Tests basic join with indexes
		w.getMovieDetails,          // Uses B-tree index on ai_myid
		w.getActorMovies,           // Uses B-tree index on ai_actor_id
		w.getMovieComments,         // Uses composite index on (ai_myid, comment_add_time)

		// Advanced analytical queries (30% of queries)
		w.testWindowFunctionRanking,   // Window functions for ranking analysis
		w.testCTERecursiveAnalysis,    // Recursive CTEs with window functions
		w.testComplexCTEAnalysis,      // Multiple CTEs with joins and aggregations
		w.testAdvancedWindowFunctions, // Advanced window function features
		w.testActorCareerAnalysisCTE,  // Complex CTE-based actor analysis
	}

	op := operations[rng.Intn(len(operations))]
	opName := ""

	switch rng.Intn(15) {
	case 0:
		opName = "indexed_title_search"
	case 1:
		opName = "indexed_rating_search"
	case 2:
		opName = "composite_index_search"
	case 3:
		opName = "unique_index_search"
	case 4:
		opName = "gin_index_search"
	case 5:
		opName = "full_table_scan"
	case 6:
		opName = "simple_join"
	case 7:
		opName = "get_movie_details"
	case 8:
		opName = "get_actor_movies"
	case 9:
		opName = "get_movie_comments"
	case 10:
		opName = "window_function_ranking"
	case 11:
		opName = "cte_recursive_analysis"
	case 12:
		opName = "complex_cte_analysis"
	case 13:
		opName = "advanced_window_functions"
	case 14:
		opName = "actor_career_cte_analysis"
	}

	return opName, op(ctx, db, rng)
}

// executeWriteOperation performs various write-heavy operations
func (w *IMDBWorkload) executeWriteOperation(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) (string, error) {
	operations := []func(context.Context, *pgxpool.Pool, *rand.Rand) error{
		w.insertNewComment,
		w.updateMovieRating,
		w.insertNewMovie,
		w.updateCommentHelpfulness,
		w.insertNewActor,
		w.addMovieActor,
		w.logMovieView,
	}

	op := operations[rng.Intn(len(operations))]
	opName := ""

	switch rng.Intn(7) {
	case 0:
		opName = "insert_comment"
	case 1:
		opName = "update_movie_rating"
	case 2:
		opName = "insert_movie"
	case 3:
		opName = "update_comment_votes"
	case 4:
		opName = "insert_actor"
	case 5:
		opName = "add_movie_actor"
	case 6:
		opName = "log_movie_view"
	}

	return opName, op(ctx, db, rng)
}

// loadSampleData populates the database with sample IMDB data using real schema
func (w *IMDBWorkload) loadSampleData(ctx context.Context, db *pgxpool.Pool, scale int) error {
	log.Printf("📊 Loading %d sample movies with Real IMDB schema...", scale)

	// Insert sample actors first (they are referenced by movies)
	actorCount := scale / 2
	log.Printf("👥 Loading %d sample actors...", actorCount)

	for i := 1; i <= actorCount; i++ {
		actorName := fmt.Sprintf("Actor %d", i)
		actorID := fmt.Sprintf("nm%07d", i)

		_, err := db.Exec(ctx, `
			INSERT INTO movies_normalized_actors (actor_id, actor_name) 
			VALUES ($1, $2)`,
			actorID, actorName)
		if err != nil {
			return fmt.Errorf("failed to insert actor %d: %w", i, err)
		}
	}

	// Insert sample movies into movies_normalized_meta
	genres := []string{"Action", "Comedy", "Drama", "Horror", "Romance", "Thriller", "Sci-Fi", "Documentary"}
	for i := 1; i <= scale; i++ {
		imdbID := fmt.Sprintf("tt%07d", i)
		title := fmt.Sprintf("Sample Movie %d", i)
		year := 1990 + (i % 35)
		rating := float64((i%100)+1) / 10.0
		country := []string{"USA", "GBR", "FRA", "DEU", "ITA", "JPN", "AUS", "CAN"}[i%8]
		overview := fmt.Sprintf("This is the overview for %s, a great %s movie from %d.", title, genres[i%len(genres)], year)

		// Create JSON metadata
		jsonData := fmt.Sprintf(`{
			"genre": "%s",
			"director": "Director %d",
			"budget": %d,
			"revenue": %d,
			"runtime": %d
		}`, genres[i%len(genres)], (i%100)+1, (i%200)*1000000, (i%500)*1000000, 90+(i%120))

		_, err := db.Exec(ctx, `
			INSERT INTO movies_normalized_meta (imdb_id, title, imdb_rating, year, country, overview, json_column, upvotes, downvotes) 
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			imdbID, title, rating, year, country, overview, jsonData, i%100, i%50)
		if err != nil {
			return fmt.Errorf("failed to insert movie %d: %w", i, err)
		}

		if i%100 == 0 {
			log.Printf("⏳ Inserted %d / %d movies...", i, scale)
		}
	}

	// Link actors to movies in movies_normalized_cast
	log.Printf("🔗 Creating movie-actor relationships...")
	for movieID := 1; movieID <= scale; movieID++ {
		// Each movie has 2-5 actors
		actorsPerMovie := 2 + (movieID % 4)
		for j := 0; j < actorsPerMovie; j++ {
			actorIdx := ((movieID + j) % actorCount) + 1
			characterName := fmt.Sprintf("Character %d", j+1)

			_, err := db.Exec(ctx, `
				INSERT INTO movies_normalized_cast (ai_myid, ai_actor_id, actor_character) 
				VALUES ($1, $2, $3) 
				ON CONFLICT DO NOTHING`,
				movieID, actorIdx, characterName)
			if err != nil {
				return fmt.Errorf("failed to link movie %d to actor %d: %w", movieID, actorIdx, err)
			}
		}
	}

	// Insert sample user comments into movies_normalized_user_comments
	commentCount := scale * 3 // 3 comments per movie on average
	log.Printf("📝 Loading %d sample user comments...", commentCount)

	commentTexts := []string{
		"Great movie! Really enjoyed it.",
		"Not bad, but could have been better.",
		"Excellent cinematography and direction.",
		"Boring and predictable.",
		"Amazing performances by the lead actors.",
		"Good story but poor execution.",
		"One of the best movies I've seen this year.",
		"Average movie. Nothing special.",
		"Terrible acting and awful script.",
		"Masterpiece! Every scene is perfect.",
	}

	for i := 1; i <= commentCount; i++ {
		movieIdx := ((i - 1) % scale) + 1
		rating := (i % 10) + 1
		commentText := commentTexts[i%len(commentTexts)]
		imdbID := fmt.Sprintf("tt%07d", movieIdx)

		_, err := db.Exec(ctx, `
			INSERT INTO movies_normalized_user_comments (ai_myid, rating, comment, imdb_id, comment_add_time) 
			VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)`,
			movieIdx, rating, commentText, imdbID)
		if err != nil {
			return fmt.Errorf("failed to insert comment %d: %w", i, err)
		}

		if i%500 == 0 {
			log.Printf("⏳ Inserted %d / %d comments...", i, commentCount)
		}
	}

	// Insert sample viewing logs
	viewLogCount := scale * 2 // 2 view records per movie on average
	log.Printf("📺 Loading %d sample viewing logs...", viewLogCount)

	for i := 1; i <= viewLogCount; i++ {
		movieIdx := ((i - 1) % scale) + 1
		imdbID := fmt.Sprintf("tt%07d", movieIdx)
		userID := (i % 1000) + 1
		timeWatchedSec := 30*60 + (i % (120 * 60)) // 30-150 minutes in seconds
		encodedData := fmt.Sprintf("session_%d_data", i)
		jsonPayload := fmt.Sprintf(`{"session_id": %d, "device": "web", "completed": %t}`, i, i%3 == 0)

		_, err := db.Exec(ctx, `
			INSERT INTO movies_viewed_logs (ai_myid, imdb_id, watched_user_id, time_watched_sec, encoded_data, json_payload) 
			VALUES ($1, $2, $3, $4, $5, $6)`,
			movieIdx, imdbID, userID, timeWatchedSec, encodedData, jsonPayload)
		if err != nil {
			return fmt.Errorf("failed to insert view log %d: %w", i, err)
		}

		if i%1000 == 0 {
			log.Printf("⏳ Inserted %d / %d view logs...", i, viewLogCount)
		}
	}

	// Insert sample voting history
	voteHistoryCount := scale * 2 // 2 vote records per movie on average
	log.Printf("🗳️ Loading %d sample voting history records...", voteHistoryCount)

	for i := 1; i <= voteHistoryCount; i++ {
		movieIdx := ((i - 1) % scale) + 1
		title := fmt.Sprintf("Sample Movie %d", movieIdx)
		imdbID := fmt.Sprintf("tt%07d", movieIdx)
		commentCount := int64(10 + (i % 500))
		upvotes := int64(i % 1000)
		downvotes := int64(i % 200)

		_, err := db.Exec(ctx, `
			INSERT INTO voting_count_history (ai_myid, title, imdb_id, comment_count, upvotes, downvotes) 
			VALUES ($1, $2, $3, $4, $5, $6)`,
			movieIdx, title, imdbID, commentCount, upvotes, downvotes)
		if err != nil {
			return fmt.Errorf("failed to insert voting history %d: %w", i, err)
		}

		if i%500 == 0 {
			log.Printf("⏳ Inserted %d / %d voting records...", i, voteHistoryCount)
		}
	}

	log.Printf("✅ Simplified IMDB sample data loaded successfully")
	return nil
}

// startRealTimeReporter provides real-time metrics during workload execution
func (w *IMDBWorkload) startRealTimeReporter(ctx context.Context, cfg *types.Config, metrics *types.Metrics, start time.Time) context.CancelFunc {
	ticker := time.NewTicker(5 * time.Second)
	reportCtx, cancel := context.WithCancel(context.Background())

	go func() {
		defer ticker.Stop()
		var lastTPS, lastQPS, lastErrors int64

		for {
			select {
			case <-ticker.C:
				// Capture current values
				tps := atomic.LoadInt64(&metrics.TPS)
				qps := atomic.LoadInt64(&metrics.QPS)
				errors := atomic.LoadInt64(&metrics.Errors)

				// Compute rates over last 5s
				tpsRate := float64(tps-lastTPS) / 5.0
				qpsRate := float64(qps-lastQPS) / 5.0
				errRate := float64(errors-lastErrors) / 5.0

				// Get elapsed time
				elapsed := time.Since(start)

				// Snapshot latencies under mutex
				metrics.Mu.Lock()
				latencies := make([]int64, len(metrics.TransactionDur))
				copy(latencies, metrics.TransactionDur)
				metrics.Mu.Unlock()

				// Compute percentiles
				p50, p95, p99 := float64(0), float64(0), float64(0)
				if len(latencies) > 0 {
					pcts := util.CalculatePercentiles(latencies, []int{50, 95, 99})
					p50 = float64(pcts[0]) / 1e6 // ns → ms
					p95 = float64(pcts[1]) / 1e6
					p99 = float64(pcts[2]) / 1e6
				}

				log.Printf("📈 REALTIME [%s] %s | TPS: %.1f | QPS: %.1f | ERR: %.1f/s | Latency: P50=%.2fms P95=%.2fms P99=%.2fms",
					elapsed.Truncate(time.Second), w.Mode, tpsRate, qpsRate, errRate, p50, p95, p99)

				// Update last values
				lastTPS = tps
				lastQPS = qps
				lastErrors = errors

			case <-reportCtx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return cancel
}

// loadFromDump loads data from a PostgreSQL dump file
func (w *IMDBWorkload) loadFromDump(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	if cfg.DataLoading.FilePath == "" {
		return fmt.Errorf("dump file path is required")
	}

	log.Printf("📦 Restoring from dump file: %s", cfg.DataLoading.FilePath)

	// Check if file exists
	if _, err := os.Stat(cfg.DataLoading.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("dump file not found: %s", cfg.DataLoading.FilePath)
	}

	// Build pg_restore command
	cmd := exec.Command("pg_restore",
		"--host", cfg.Database.Host,
		"--port", fmt.Sprintf("%d", cfg.Database.Port),
		"--username", cfg.Database.Username,
		"--dbname", cfg.Database.Dbname,
		"--no-password",
		"--clean",
		"--if-exists",
		"--data-only",
		"--verbose",
		cfg.DataLoading.FilePath)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.Database.Password))

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("🔄 Running pg_restore command...")
	err := cmd.Run()

	if err != nil {
		log.Printf("❌ pg_restore stderr: %s", stderr.String())
		return fmt.Errorf("pg_restore failed: %w", err)
	}

	log.Printf("✅ Dump file loaded successfully")
	if stdout.Len() > 0 {
		log.Printf("📝 pg_restore output: %s", stdout.String())
	}

	return nil
}

// loadFromSQL loads data from an SQL file
func (w *IMDBWorkload) loadFromSQL(ctx context.Context, db *pgxpool.Pool, cfg *types.Config) error {
	if cfg.DataLoading.FilePath == "" {
		return fmt.Errorf("SQL file path is required")
	}

	log.Printf("📜 Loading from SQL file: %s", cfg.DataLoading.FilePath)

	// Check if file exists
	if _, err := os.Stat(cfg.DataLoading.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("SQL file not found: %s", cfg.DataLoading.FilePath)
	}

	// Read the SQL file
	sqlContent, err := os.ReadFile(cfg.DataLoading.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %w", err)
	}

	// Split SQL commands (basic implementation)
	// This handles simple cases; for complex SQL files, consider using a proper SQL parser
	sqlStatements := strings.Split(string(sqlContent), ";")

	log.Printf("🔄 Executing %d SQL statements...", len(sqlStatements))

	successCount := 0
	for i, statement := range sqlStatements {
		statement = strings.TrimSpace(statement)
		if statement == "" || strings.HasPrefix(statement, "--") {
			continue
		}

		_, err := db.Exec(ctx, statement)
		if err != nil {
			log.Printf("⚠️  Warning: Failed to execute statement %d: %v", i+1, err)
			log.Printf("📄 Statement: %s", statement[:min(100, len(statement))])
		} else {
			successCount++
		}

		// Progress update every 100 statements
		if (i+1)%100 == 0 {
			log.Printf("⏳ Processed %d / %d statements...", i+1, len(sqlStatements))
		}
	}

	log.Printf("✅ SQL file loaded successfully: %d statements executed", successCount)
	return nil
}

// min helper function for string truncation
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
