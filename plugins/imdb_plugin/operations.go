// internal/workload/imdb/operations.go
// Database operations using simplified IMDB schema structure
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// READ OPERATIONS

// searchMoviesByGenre finds movies by genre using simplified schema
func (w *IMDBWorkload) searchMoviesByGenre(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Extract genre from JSON column
	genres := []string{"Action", "Comedy", "Drama", "Horror", "Romance", "Thriller", "Sci-Fi", "Documentary"}
	genre := genres[rng.Intn(len(genres))]

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE json_column->>'genre' = $1 AND imdb_rating IS NOT NULL 
		ORDER BY imdb_rating DESC LIMIT 20`,
		genre)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year int
		var rating *float64
		if err := rows.Scan(&imdbID, &title, &year, &rating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getMovieDetails retrieves detailed information about a movie using simplified schema
func (w *IMDBWorkload) getMovieDetails(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1

	// Get movie info from movies_normalized_meta
	var title, country, overview string
	var year int
	var rating *float64
	var jsonColumn string

	err := db.QueryRow(ctx, `
		SELECT title, year, country, overview, imdb_rating, json_column 
		FROM movies_normalized_meta 
		WHERE ai_myid = $1`,
		movieIdx).Scan(&title, &year, &country, &overview, &rating, &jsonColumn)
	if err != nil {
		return err
	}

	// Get cast information
	rows, err := db.Query(ctx, `
		SELECT a.actor_name, c.actor_character 
		FROM movies_normalized_cast c
		JOIN movies_normalized_actors a ON c.ai_actor_id = a.ai_actor_id
		WHERE c.ai_myid = $1 
		LIMIT 10`,
		movieIdx)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var actorName, characterName string
		if err := rows.Scan(&actorName, &characterName); err != nil {
			return err
		}
	}

	return rows.Err()
}

// getTopRatedMovies retrieves highest rated movies using simplified schema
func (w *IMDBWorkload) getTopRatedMovies(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minRating := float64(rng.Intn(50)+50) / 10.0 // 5.0-10.0 rating

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating, upvotes 
		FROM movies_normalized_meta 
		WHERE imdb_rating >= $1 
		ORDER BY imdb_rating DESC, upvotes DESC 
		LIMIT 25`,
		minRating)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year, upvotes int
		var rating *float64
		if err := rows.Scan(&imdbID, &title, &year, &rating, &upvotes); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getActorMovies finds movies for a specific actor using simplified schema
func (w *IMDBWorkload) getActorMovies(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	actorIdx := rng.Intn(500) + 1

	rows, err := db.Query(ctx, `
		SELECT m.imdb_id, m.title, m.year, c.actor_character 
		FROM movies_normalized_cast c
		JOIN movies_normalized_meta m ON c.ai_myid = m.ai_myid
		WHERE c.ai_actor_id = $1 
		ORDER BY m.year DESC 
		LIMIT 20`,
		actorIdx)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title, characterName string
		var year int
		if err := rows.Scan(&imdbID, &title, &year, &characterName); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getMovieComments retrieves user comments for a movie using simplified schema
func (w *IMDBWorkload) getMovieComments(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		SELECT rating, comment, comment_add_time 
		FROM movies_normalized_user_comments 
		WHERE ai_myid = $1 
		ORDER BY comment_add_time DESC 
		LIMIT 15`,
		movieIdx)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var rating int
		var comment string
		var commentTime time.Time
		if err := rows.Scan(&rating, &comment, &commentTime); err != nil {
			return err
		}
	}
	return rows.Err()
}

// searchMoviesByYear finds movies from a specific year range using simplified schema
func (w *IMDBWorkload) searchMoviesByYear(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	startYear := 1990 + rng.Intn(35)
	endYear := startYear + 5

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE year BETWEEN $1 AND $2 
		AND imdb_rating IS NOT NULL 
		ORDER BY imdb_rating DESC 
		LIMIT 30`,
		startYear, endYear)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year int
		var rating *float64
		if err := rows.Scan(&imdbID, &title, &year, &rating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getRecentViewingActivity retrieves recent viewing logs using simplified schema
func (w *IMDBWorkload) getRecentViewingActivity(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	userID := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		SELECT v.imdb_id, m.title, v.watched_time, v.time_watched_sec, v.json_payload 
		FROM movies_viewed_logs v
		JOIN movies_normalized_meta m ON v.ai_myid = m.ai_myid
		WHERE v.watched_user_id = $1 
		ORDER BY v.watched_time DESC 
		LIMIT 20`,
		userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title, jsonPayload string
		var timeWatchedSec int
		var watchedTime time.Time
		if err := rows.Scan(&imdbID, &title, &watchedTime, &timeWatchedSec, &jsonPayload); err != nil {
			return err
		}
	}
	return rows.Err()
}

// searchMoviesByTitle searches movies by title pattern (uses idx_nmm_title index)
func (w *IMDBWorkload) searchMoviesByTitle(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Generate partial title search to test LIKE with index
	searchTerms := []string{"The", "Love", "War", "Night", "Day", "Last", "First", "New", "Big", "Life"}
	searchTerm := searchTerms[rng.Intn(len(searchTerms))] + "%"

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE title LIKE $1 
		ORDER BY imdb_rating DESC NULLS LAST 
		LIMIT 25`,
		searchTerm)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year int
		var rating *float64
		if err := rows.Scan(&imdbID, &title, &year, &rating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// searchMoviesByCountryYear searches using composite index (country, year, imdb_rating)
func (w *IMDBWorkload) searchMoviesByCountryYear(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	countries := []string{"USA", "UK", "Canada", "France", "Germany", "Japan", "Australia", "Italy"}
	country := countries[rng.Intn(len(countries))]
	year := 1990 + rng.Intn(35)

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, country, imdb_rating 
		FROM movies_normalized_meta 
		WHERE country = $1 AND year = $2 AND imdb_rating IS NOT NULL
		ORDER BY imdb_rating DESC 
		LIMIT 20`,
		country, year)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title, country string
		var year int
		var rating *float64
		if err := rows.Scan(&imdbID, &title, &year, &country, &rating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getMovieStatsByIMDbID uses unique index on imdb_id
func (w *IMDBWorkload) getMovieStatsByIMDbID(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieID := rng.Intn(10000) + 1
	imdbID := fmt.Sprintf("tt%07d", movieID)

	var title, country string
	var year, upvotes, downvotes int
	var rating *float64

	err := db.QueryRow(ctx, `
		SELECT title, year, country, imdb_rating, upvotes, downvotes 
		FROM movies_normalized_meta 
		WHERE imdb_id = $1`,
		imdbID).Scan(&title, &year, &country, &rating, &upvotes, &downvotes)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	return nil
}

// searchMoviesFullTableScan forces full table scan by searching unindexed overview column
func (w *IMDBWorkload) searchMoviesFullTableScan(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	// Search in overview column which has no index - forces full table scan
	searchTerms := []string{"story", "life", "world", "family", "love", "war", "journey", "adventure"}
	searchTerm := "%" + searchTerms[rng.Intn(len(searchTerms))] + "%"

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, overview 
		FROM movies_normalized_meta 
		WHERE overview ILIKE $1 
		LIMIT 15`,
		searchTerm)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title, overview string
		var year int
		if err := rows.Scan(&imdbID, &title, &year, &overview); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getComplexMovieAnalytics performs complex multi-table join with aggregations
func (w *IMDBWorkload) getComplexMovieAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minRating := float64(rng.Intn(30)+50) / 10.0 // 5.0-8.0 rating

	rows, err := db.Query(ctx, `
		SELECT 
			m.imdb_id,
			m.title,
			m.year,
			m.imdb_rating,
			COUNT(DISTINCT c.ai_actor_id) as actor_count,
			COUNT(DISTINCT uc.comment_id) as comment_count,
			AVG(uc.rating::float) as avg_user_rating,
			COUNT(DISTINCT vl.watched_user_id) as unique_viewers
		FROM movies_normalized_meta m
		LEFT JOIN movies_normalized_cast c ON m.ai_myid = c.ai_myid
		LEFT JOIN movies_normalized_user_comments uc ON m.ai_myid = uc.ai_myid
		LEFT JOIN movies_viewed_logs vl ON m.ai_myid = vl.ai_myid
		WHERE m.imdb_rating >= $1
		GROUP BY m.ai_myid, m.imdb_id, m.title, m.year, m.imdb_rating
		HAVING COUNT(DISTINCT c.ai_actor_id) > 0
		ORDER BY m.imdb_rating DESC, unique_viewers DESC
		LIMIT 10`,
		minRating)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year, actorCount, commentCount, uniqueViewers int
		var imdbRating *float64
		var avgUserRating *float64
		if err := rows.Scan(&imdbID, &title, &year, &imdbRating, &actorCount, &commentCount, &avgUserRating, &uniqueViewers); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getActorAnalytics performs complex actor analysis with multiple joins
func (w *IMDBWorkload) getActorAnalytics(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minMovieCount := rng.Intn(5) + 2 // Actors with at least 2-7 movies

	rows, err := db.Query(ctx, `
		SELECT 
			a.actor_name,
			COUNT(DISTINCT c.ai_myid) as movie_count,
			AVG(m.imdb_rating) as avg_movie_rating,
			MAX(m.imdb_rating) as best_movie_rating,
			MIN(m.year) as career_start_year,
			MAX(m.year) as latest_movie_year,
			(SELECT STRING_AGG(title, ', ') 
			 FROM (
				SELECT DISTINCT m2.title 
				FROM movies_normalized_meta m2
				JOIN movies_normalized_cast c2 ON m2.ai_myid = c2.ai_myid
				WHERE c2.ai_actor_id = a.ai_actor_id 
				AND m2.imdb_rating IS NOT NULL
				ORDER BY m2.imdb_rating DESC 
				LIMIT 3
			 ) as top_movies_subq
			) as top_movies
		FROM movies_normalized_actors a
		JOIN movies_normalized_cast c ON a.ai_actor_id = c.ai_actor_id
		JOIN movies_normalized_meta m ON c.ai_myid = m.ai_myid
		WHERE m.imdb_rating IS NOT NULL
		GROUP BY a.ai_actor_id, a.actor_name
		HAVING COUNT(DISTINCT c.ai_myid) >= $1
		ORDER BY avg_movie_rating DESC NULLS LAST, movie_count DESC
		LIMIT 15`,
		minMovieCount)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var actorName string
		var topMovies *string
		var movieCount, careerStartYear, latestMovieYear int
		var avgMovieRating, bestMovieRating *float64
		if err := rows.Scan(&actorName, &movieCount, &avgMovieRating, &bestMovieRating, &careerStartYear, &latestMovieYear, &topMovies); err != nil {
			return err
		}
	}
	return rows.Err()
}

// getMovieTrendsAnalysis performs time-series analysis of movie trends
func (w *IMDBWorkload) getMovieTrendsAnalysis(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	startYear := 1980 + rng.Intn(30)
	endYear := startYear + 15

	rows, err := db.Query(ctx, `
		SELECT 
			year,
			COUNT(*) as total_movies,
			AVG(imdb_rating) as avg_rating,
			MAX(imdb_rating) as best_rating,
			COUNT(*) FILTER (WHERE imdb_rating >= 8.0) as highly_rated_count,
			AVG(upvotes) as avg_popularity
		FROM movies_normalized_meta
		WHERE year BETWEEN $1 AND $2 
		AND imdb_rating IS NOT NULL
		GROUP BY year
		ORDER BY year DESC`,
		startYear, endYear)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var year, totalMovies, highlyRatedCount int
		var avgRating, bestRating, avgPopularity *float64
		if err := rows.Scan(&year, &totalMovies, &avgRating, &bestRating, &highlyRatedCount, &avgPopularity); err != nil {
			return err
		}
	}
	return rows.Err()
}

// WRITE OPERATIONS

// insertNewComment adds a new user comment using simplified schema
func (w *IMDBWorkload) insertNewComment(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1
	rating := rng.Intn(10) + 1

	reviewTexts := []string{
		"Amazing cinematography and stellar performances!",
		"A bit slow paced but worth watching.",
		"Excellent direction and compelling storyline.",
		"Not what I expected, but pleasantly surprised.",
		"Outstanding acting throughout the entire film.",
		"Could have been better with a tighter script.",
		"Visually stunning with great character development.",
		"Typical Hollywood fare, nothing groundbreaking.",
		"Brilliant storytelling and emotional depth.",
		"Great entertainment value for the whole family.",
	}
	reviewText := reviewTexts[rng.Intn(len(reviewTexts))]

	_, err := db.Exec(ctx, `
		INSERT INTO movies_normalized_user_comments (ai_myid, rating, comment, comment_add_time) 
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)`,
		movieIdx, rating, reviewText)
	return err
}

// updateMovieRating updates rating statistics using simplified schema
func (w *IMDBWorkload) updateMovieRating(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1
	newRating := float64(rng.Intn(100)+1) / 10.0
	voteIncrement := rng.Intn(50) + 1

	_, err := db.Exec(ctx, `
		UPDATE movies_normalized_meta 
		SET imdb_rating = $2, upvotes = upvotes + $3 
		WHERE ai_myid = $1`,
		movieIdx, newRating, voteIncrement)
	return err
}

// insertNewMovie adds a new movie to the database using simplified schema
func (w *IMDBWorkload) insertNewMovie(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	newID := rng.Intn(1000000) + 100000
	imdbID := fmt.Sprintf("tt%07d", newID)
	title := fmt.Sprintf("New Movie %d", newID)
	year := 2020 + rng.Intn(5)
	genres := []string{"Action", "Comedy", "Drama", "Horror", "Romance", "Thriller", "Sci-Fi", "Documentary"}
	genre := genres[rng.Intn(len(genres))]
	country := "USA"
	overview := fmt.Sprintf("This is the overview for %s, a great %s movie.", title, genre)

	jsonData := fmt.Sprintf(`{
		"genre": "%s",
		"director": "New Director %d",
		"budget": %d,
		"revenue": %d,
		"runtime": %d
	}`, genre, rng.Intn(100)+1, rng.Intn(200)*1000000, rng.Intn(500)*1000000, 90+rng.Intn(120))

	_, err := db.Exec(ctx, `
		INSERT INTO movies_normalized_meta (imdb_id, title, year, country, overview, json_column, upvotes, downvotes) 
		VALUES ($1, $2, $3, $4, $5, $6, 0, 0)`,
		imdbID, title, year, country, overview, jsonData)
	return err
}

// updateCommentHelpfulness updates a comment using simplified schema
func (w *IMDBWorkload) updateCommentHelpfulness(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1

	// Update a random comment for this movie
	_, err := db.Exec(ctx, `
		UPDATE movies_normalized_user_comments 
		SET comment_update_time = CURRENT_TIMESTAMP
		WHERE ai_myid = $1 
		AND comment_id = (
			SELECT comment_id FROM movies_normalized_user_comments 
			WHERE ai_myid = $1 
			ORDER BY RANDOM() 
			LIMIT 1
		)`,
		movieIdx)
	return err
}

// insertNewActor adds a new actor to the database using simplified schema
func (w *IMDBWorkload) insertNewActor(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	newID := rng.Intn(1000000) + 100000
	actorID := fmt.Sprintf("nm%07d", newID)
	name := fmt.Sprintf("New Actor %d", newID)

	_, err := db.Exec(ctx, `
		INSERT INTO movies_normalized_actors (actor_id, actor_name) 
		VALUES ($1, $2)`,
		actorID, name)
	return err
}

// addMovieActor links an actor to a movie using simplified schema
func (w *IMDBWorkload) addMovieActor(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1
	actorIdx := rng.Intn(500) + 1
	character := fmt.Sprintf("Character %d", rng.Intn(10)+1)

	_, err := db.Exec(ctx, `
		INSERT INTO movies_normalized_cast (ai_myid, ai_actor_id, actor_character) 
		VALUES ($1, $2, $3) 
		ON CONFLICT DO NOTHING`,
		movieIdx, actorIdx, character)
	return err
}

// logMovieView records a movie viewing event using simplified schema
func (w *IMDBWorkload) logMovieView(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1
	imdbID := fmt.Sprintf("tt%07d", movieIdx)
	userID := rng.Intn(1000) + 1
	timeWatchedSec := rng.Intn(180*60) + 30*60 // 30-210 minutes in seconds
	encodedData := fmt.Sprintf("session_%d_data", rng.Intn(10000))
	jsonPayload := fmt.Sprintf(`{"session_id": %d, "device": "web", "completed": %t}`, rng.Intn(10000), rng.Intn(3) == 0)

	_, err := db.Exec(ctx, `
		INSERT INTO movies_viewed_logs (ai_myid, imdb_id, watched_user_id, time_watched_sec, encoded_data, json_payload) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		movieIdx, imdbID, userID, timeWatchedSec, encodedData, jsonPayload)
	return err
}
