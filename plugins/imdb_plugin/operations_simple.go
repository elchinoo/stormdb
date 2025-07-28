package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Simple operations for testing index utilization without complex aggregations

// testIndexedTitleSearch tests B-tree index on title
func (w *IMDBWorkload) testIndexedTitleSearch(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	searchTerms := []string{"The", "Love", "War", "Night", "Day", "Last", "First", "New", "Big", "Life"}
	searchTerm := searchTerms[rng.Intn(len(searchTerms))] + "%"

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE title LIKE $1 
		LIMIT 10`,
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

// testIndexedRatingSearch tests B-tree index on imdb_rating
func (w *IMDBWorkload) testIndexedRatingSearch(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minRating := float64(rng.Intn(40)+60) / 10.0 // 6.0-10.0 rating

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE imdb_rating >= $1 
		LIMIT 15`,
		minRating)
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

// testCompositeIndexSearch tests composite index on (country, year, imdb_rating)
func (w *IMDBWorkload) testCompositeIndexSearch(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	countries := []string{"USA", "UK", "Canada", "France", "Germany", "Japan", "Australia", "Italy"}
	country := countries[rng.Intn(len(countries))]
	year := 1990 + rng.Intn(35)

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, country, imdb_rating 
		FROM movies_normalized_meta 
		WHERE country = $1 AND year = $2 
		LIMIT 10`,
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

// testUniqueIndexSearch tests unique index on imdb_id
func (w *IMDBWorkload) testUniqueIndexSearch(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieID := rng.Intn(10000) + 1
	imdbID := fmt.Sprintf("tt%07d", movieID)

	var title, country string
	var year int
	var rating *float64

	err := db.QueryRow(ctx, `
		SELECT title, year, country, imdb_rating 
		FROM movies_normalized_meta 
		WHERE imdb_id = $1`,
		imdbID).Scan(&title, &year, &country, &rating)
	if err != nil {
		// Expected for non-existent movies
		return nil
	}
	return nil
}

// testGINIndexSearch tests GIN index on json_column
func (w *IMDBWorkload) testGINIndexSearch(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	genres := []string{"Action", "Comedy", "Drama", "Horror", "Romance", "Thriller", "Sci-Fi", "Documentary"}
	genre := genres[rng.Intn(len(genres))]

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year, imdb_rating 
		FROM movies_normalized_meta 
		WHERE json_column->>'genre' = $1 
		LIMIT 10`,
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

// testFullTableScan forces full table scan on unindexed column
func (w *IMDBWorkload) testFullTableScan(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	searchTerms := []string{"story", "life", "world", "family", "love", "war", "journey", "adventure"}
	searchTerm := "%" + searchTerms[rng.Intn(len(searchTerms))] + "%"

	rows, err := db.Query(ctx, `
		SELECT imdb_id, title, year 
		FROM movies_normalized_meta 
		WHERE overview ILIKE $1 
		LIMIT 5`,
		searchTerm)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year int
		if err := rows.Scan(&imdbID, &title, &year); err != nil {
			return err
		}
	}
	return rows.Err()
}

// testSimpleJoin tests basic join performance with indexes
func (w *IMDBWorkload) testSimpleJoin(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	movieIdx := rng.Intn(1000) + 1

	rows, err := db.Query(ctx, `
		SELECT a.actor_name, c.actor_character 
		FROM movies_normalized_cast c
		JOIN movies_normalized_actors a ON c.ai_actor_id = a.ai_actor_id
		WHERE c.ai_myid = $1 
		LIMIT 5`,
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

// testWindowFunctionRanking uses window functions for movie ranking analysis
func (w *IMDBWorkload) testWindowFunctionRanking(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	countries := []string{"USA", "UK", "Canada", "France", "Germany", "Japan", "Australia", "Italy"}
	country := countries[rng.Intn(len(countries))]

	rows, err := db.Query(ctx, `
		SELECT 
			imdb_id,
			title,
			year,
			imdb_rating,
			ROW_NUMBER() OVER (ORDER BY imdb_rating DESC NULLS LAST) as rating_rank,
			RANK() OVER (PARTITION BY year ORDER BY imdb_rating DESC NULLS LAST) as year_rank,
			DENSE_RANK() OVER (ORDER BY upvotes DESC) as popularity_rank,
			LAG(imdb_rating) OVER (ORDER BY year, imdb_rating DESC) as prev_rating,
			LEAD(imdb_rating) OVER (ORDER BY year, imdb_rating DESC) as next_rating,
			AVG(imdb_rating) OVER (PARTITION BY year) as year_avg_rating,
			PERCENT_RANK() OVER (ORDER BY imdb_rating) as rating_percentile
		FROM movies_normalized_meta 
		WHERE country = $1 AND imdb_rating IS NOT NULL
		ORDER BY imdb_rating DESC
		LIMIT 20`,
		country)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var imdbID, title string
		var year, ratingRank, yearRank, popularityRank int
		var imdbRating, prevRating, nextRating, yearAvgRating, ratingPercentile *float64
		if err := rows.Scan(&imdbID, &title, &year, &imdbRating, &ratingRank, &yearRank, &popularityRank, &prevRating, &nextRating, &yearAvgRating, &ratingPercentile); err != nil {
			return err
		}
	}
	return rows.Err()
}

// testCTERecursiveAnalysis uses CTEs for hierarchical/recursive analysis
func (w *IMDBWorkload) testCTERecursiveAnalysis(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minYear := 2000 + rng.Intn(20)

	rows, err := db.Query(ctx, `
		WITH RECURSIVE year_series AS (
			-- Base case: start year
			SELECT $1 as year_num, 0 as level
			UNION ALL
			-- Recursive case: generate next 5 years
			SELECT year_num + 1, level + 1
			FROM year_series
			WHERE level < 5
		),
		movie_stats AS (
			SELECT 
				m.year,
				COUNT(*) as movie_count,
				AVG(m.imdb_rating) as avg_rating,
				MAX(m.imdb_rating) as max_rating,
				MIN(m.imdb_rating) as min_rating,
				STDDEV(m.imdb_rating) as rating_stddev
			FROM movies_normalized_meta m
			WHERE m.year >= $1 AND m.imdb_rating IS NOT NULL
			GROUP BY m.year
		)
		SELECT 
			ys.year_num,
			COALESCE(ms.movie_count, 0) as movie_count,
			COALESCE(ms.avg_rating, 0) as avg_rating,
			COALESCE(ms.max_rating, 0) as max_rating,
			COALESCE(ms.min_rating, 0) as min_rating,
			COALESCE(ms.rating_stddev, 0) as rating_stddev,
			-- Running totals using window functions over CTE
			SUM(COALESCE(ms.movie_count, 0)) OVER (ORDER BY ys.year_num) as cumulative_movies,
			AVG(COALESCE(ms.avg_rating, 0)) OVER (ORDER BY ys.year_num ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) as moving_avg_rating
		FROM year_series ys
		LEFT JOIN movie_stats ms ON ys.year_num = ms.year
		ORDER BY ys.year_num`,
		minYear)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var yearNum, movieCount, cumulativeMovies int
		var avgRating, maxRating, minRating, ratingStddev, movingAvgRating float64
		if err := rows.Scan(&yearNum, &movieCount, &avgRating, &maxRating, &minRating, &ratingStddev, &cumulativeMovies, &movingAvgRating); err != nil {
			return err
		}
	}
	return rows.Err()
}

// testComplexCTEAnalysis uses multiple CTEs for comprehensive movie analysis
func (w *IMDBWorkload) testComplexCTEAnalysis(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minRating := float64(rng.Intn(30)+60) / 10.0 // 6.0-9.0 rating

	rows, err := db.Query(ctx, `
		WITH top_movies AS (
			SELECT ai_myid, imdb_id, title, year, imdb_rating, upvotes
			FROM movies_normalized_meta
			WHERE imdb_rating >= $1
		),
		movie_cast_counts AS (
			SELECT 
				tm.ai_myid,
				tm.title,
				tm.imdb_rating,
				COUNT(c.ai_actor_id) as cast_size
			FROM top_movies tm
			LEFT JOIN movies_normalized_cast c ON tm.ai_myid = c.ai_myid
			GROUP BY tm.ai_myid, tm.title, tm.imdb_rating
		),
		movie_comment_stats AS (
			SELECT 
				tm.ai_myid,
				COUNT(uc.comment_id) as comment_count,
				AVG(uc.rating::float) as avg_user_rating,
				MAX(uc.comment_add_time) as latest_comment
			FROM top_movies tm
			LEFT JOIN movies_normalized_user_comments uc ON tm.ai_myid = uc.ai_myid
			GROUP BY tm.ai_myid
		),
		movie_popularity AS (
			SELECT 
				tm.ai_myid,
				COUNT(DISTINCT vl.watched_user_id) as unique_viewers,
				AVG(vl.time_watched_sec) as avg_watch_time
			FROM top_movies tm
			LEFT JOIN movies_viewed_logs vl ON tm.ai_myid = vl.ai_myid
			GROUP BY tm.ai_myid
		)
		SELECT 
			tm.title,
			tm.year,
			tm.imdb_rating,
			mcc.cast_size,
			mcs.comment_count,
			mcs.avg_user_rating,
			mp.unique_viewers,
			mp.avg_watch_time,
			-- Window functions for ranking and percentiles
			NTILE(4) OVER (ORDER BY tm.imdb_rating) as rating_quartile,
			PERCENT_RANK() OVER (ORDER BY mcc.cast_size) as cast_size_percentile,
			ROW_NUMBER() OVER (PARTITION BY tm.year ORDER BY tm.upvotes DESC) as year_popularity_rank
		FROM top_movies tm
		JOIN movie_cast_counts mcc ON tm.ai_myid = mcc.ai_myid
		LEFT JOIN movie_comment_stats mcs ON tm.ai_myid = mcs.ai_myid
		LEFT JOIN movie_popularity mp ON tm.ai_myid = mp.ai_myid
		WHERE mcc.cast_size > 0
		ORDER BY tm.imdb_rating DESC, tm.upvotes DESC
		LIMIT 15`,
		minRating)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var title string
		var year, castSize, commentCount, uniqueViewers, ratingQuartile, yearPopularityRank int
		var imdbRating, avgUserRating, avgWatchTime, castSizePercentile *float64
		if err := rows.Scan(&title, &year, &imdbRating, &castSize, &commentCount, &avgUserRating, &uniqueViewers, &avgWatchTime, &ratingQuartile, &castSizePercentile, &yearPopularityRank); err != nil {
			return err
		}
	}
	return rows.Err()
}

// testAdvancedWindowFunctions uses advanced window function features
func (w *IMDBWorkload) testAdvancedWindowFunctions(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	startYear := 1990 + rng.Intn(20)
	endYear := startYear + 10

	rows, err := db.Query(ctx, `
		WITH movie_data AS (
			SELECT 
				imdb_id,
				title,
				year,
				imdb_rating,
				upvotes,
				json_column->>'genre' as genre
			FROM movies_normalized_meta
			WHERE year BETWEEN $1 AND $2 AND imdb_rating IS NOT NULL
		)
		SELECT 
			title,
			year,
			genre,
			imdb_rating,
			upvotes,
			-- Ranking functions
			ROW_NUMBER() OVER w as row_num,
			RANK() OVER w as rank_pos,
			DENSE_RANK() OVER w as dense_rank_pos,
			-- Value functions
			FIRST_VALUE(title) OVER w as best_movie_title,
			LAST_VALUE(title) OVER (PARTITION BY genre ORDER BY imdb_rating RANGE BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as worst_movie_title,
			NTH_VALUE(title, 2) OVER w as second_best_title,
			-- Offset functions
			LAG(imdb_rating, 1, 0) OVER (PARTITION BY genre ORDER BY year) as prev_year_rating,
			LEAD(imdb_rating, 1, 0) OVER (PARTITION BY genre ORDER BY year) as next_year_rating,
			-- Aggregate functions as window functions
			SUM(upvotes) OVER (PARTITION BY genre ORDER BY year ROWS BETWEEN 1 PRECEDING AND 1 FOLLOWING) as rolling_popularity,
			AVG(imdb_rating) OVER (PARTITION BY genre ORDER BY year ROWS BETWEEN 2 PRECEDING AND CURRENT ROW) as rolling_avg_rating,
			-- Statistical functions
			PERCENT_RANK() OVER w as rating_percent_rank,
			CUME_DIST() OVER w as rating_cumulative_dist,
			NTILE(5) OVER w as rating_quintile
		FROM movie_data
		WHERE genre IS NOT NULL
		WINDOW w AS (PARTITION BY genre ORDER BY imdb_rating DESC)
		ORDER BY genre, imdb_rating DESC
		LIMIT 25`,
		startYear, endYear)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var title, genre, bestMovieTitle, worstMovieTitle string
		var secondBestTitle *string
		var year, upvotes, rowNum, rankPos, denseRankPos, ratingQuintile int
		var imdbRating, prevYearRating, nextYearRating, rollingPopularity, rollingAvgRating, ratingPercentRank, ratingCumulativeDist float64
		if err := rows.Scan(&title, &year, &genre, &imdbRating, &upvotes, &rowNum, &rankPos, &denseRankPos, &bestMovieTitle, &worstMovieTitle, &secondBestTitle, &prevYearRating, &nextYearRating, &rollingPopularity, &rollingAvgRating, &ratingPercentRank, &ratingCumulativeDist, &ratingQuintile); err != nil {
			return err
		}
	}
	return rows.Err()
}

// testActorCareerAnalysisCTE uses CTEs and window functions for actor career analysis
func (w *IMDBWorkload) testActorCareerAnalysisCTE(ctx context.Context, db *pgxpool.Pool, rng *rand.Rand) error {
	minMovies := rng.Intn(3) + 3 // Actors with at least 3-6 movies

	rows, err := db.Query(ctx, `
		WITH actor_movies AS (
			SELECT 
				a.ai_actor_id,
				a.actor_name,
				m.ai_myid,
				m.title,
				m.year,
				m.imdb_rating,
				c.actor_character,
				ROW_NUMBER() OVER (PARTITION BY a.ai_actor_id ORDER BY m.year) as career_movie_number
			FROM movies_normalized_actors a
			JOIN movies_normalized_cast c ON a.ai_actor_id = c.ai_actor_id
			JOIN movies_normalized_meta m ON c.ai_myid = m.ai_myid
			WHERE m.imdb_rating IS NOT NULL
		),
		actor_career_base AS (
			SELECT 
				ai_actor_id,
				actor_name,
				COUNT(*) as total_movies,
				MIN(year) as career_start,
				MAX(year) as career_end,
				AVG(imdb_rating) as avg_movie_rating,
				MAX(imdb_rating) as best_movie_rating,
				MIN(imdb_rating) as worst_movie_rating
			FROM actor_movies
			GROUP BY ai_actor_id, actor_name
			HAVING COUNT(*) >= $1
		),
		actor_debut_latest AS (
			SELECT DISTINCT
				am.ai_actor_id,
				FIRST_VALUE(am.title) OVER (PARTITION BY am.ai_actor_id ORDER BY am.year) as debut_movie,
				LAST_VALUE(am.title) OVER (PARTITION BY am.ai_actor_id ORDER BY am.year ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as latest_movie
			FROM actor_movies am
			WHERE am.ai_actor_id IN (SELECT ai_actor_id FROM actor_career_base)
		)
		SELECT 
			acb.actor_name,
			acb.total_movies,
			acb.career_start,
			acb.career_end,
			acb.career_end - acb.career_start as career_span,
			acb.avg_movie_rating,
			acb.best_movie_rating,
			acb.worst_movie_rating,
			adl.debut_movie,
			adl.latest_movie,
			-- Ranking among all actors
			RANK() OVER (ORDER BY acb.avg_movie_rating DESC) as avg_rating_rank,
			RANK() OVER (ORDER BY acb.total_movies DESC) as productivity_rank,
			NTILE(10) OVER (ORDER BY acb.best_movie_rating DESC) as peak_performance_decile
		FROM actor_career_base acb
		JOIN actor_debut_latest adl ON acb.ai_actor_id = adl.ai_actor_id
		ORDER BY acb.avg_movie_rating DESC
		LIMIT 20`,
		minMovies)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var actorName, debutMovie, latestMovie string
		var totalMovies, careerStart, careerEnd, careerSpan, avgRatingRank, productivityRank, peakPerformanceDecile int
		var avgMovieRating, bestMovieRating, worstMovieRating *float64
		if err := rows.Scan(&actorName, &totalMovies, &careerStart, &careerEnd, &careerSpan, &avgMovieRating, &bestMovieRating, &worstMovieRating, &debutMovie, &latestMovie, &avgRatingRank, &productivityRank, &peakPerformanceDecile); err != nil {
			return err
		}
	}
	return rows.Err()
}
