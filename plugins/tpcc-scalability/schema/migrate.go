package schema

import (
	"context"
	"database/sql"
	"io/ioutil"
	"path/filepath"
	"sort"
)

// Migrate applies all SQL migration files in numeric order.
func Migrate(ctx context.Context, db *sql.DB, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		sqlBytes, err := ioutil.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return err
		}
	}
	return nil
}
