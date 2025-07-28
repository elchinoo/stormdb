// internal/database/postgres.go
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/elchinoo/stormdb/pkg/types"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func NewPostgres(cfg *types.Config) (*Postgres, error) {
	dsn := fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s pool_max_conns=%d pool_min_conns=%d pool_max_conn_lifetime=1h pool_max_conn_idle_time=30m pool_health_check_period=1m connect_timeout=10",
		cfg.Database.Username, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Dbname, cfg.Database.Sslmode,
		cfg.Connections, cfg.Connections/2, // min connections = half of max
	)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Postgres{Pool: pool}, nil
}

func (p *Postgres) SetupTestTable(scale int) error {
	ctx := context.Background()

	_, err := p.Pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS loadtest (
            id BIGINT PRIMARY KEY,
            val TEXT,
            updated TIMESTAMPTZ
        )`)
	if err != nil {
		return err
	}

	var count int64
	err = p.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM loadtest").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		fmt.Printf("Seeding %d rows into loadtest...\n", scale)
		for i := 1; i <= scale; i++ {
			_, err := p.Pool.Exec(ctx,
				"INSERT INTO loadtest (id, val, updated) VALUES ($1, $2, NOW()) ON CONFLICT (id) DO NOTHING",
				i, fmt.Sprintf("initial_%d", i))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *Postgres) Close() {
	p.Pool.Close()
}

// BuildConnectionString creates a connection string from config for single connections
func BuildConnectionString(cfg *types.Config) string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%d dbname=%s sslmode=%s connect_timeout=10",
		cfg.Database.Username, cfg.Database.Password,
		cfg.Database.Host, cfg.Database.Port,
		cfg.Database.Dbname, cfg.Database.Sslmode,
	)
}
