package migrations

import (
	"database/sql"
	"fmt"
)

// RunAll applies any pending migrations to db in order.  A schema_migrations
// tracking table is created on first run; already-applied migrations are
// skipped so the function is safe to call on every startup.
func RunAll(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	all, err := All()
	if err != nil {
		return err
	}

	for _, m := range all {
		var count int
		_ = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE name = $1", m.Name).Scan(&count)
		if count > 0 {
			continue // already applied
		}
		if _, err := db.Exec(m.Up); err != nil {
			return fmt.Errorf("apply %s: %w", m.Name, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (name) VALUES ($1)", m.Name); err != nil {
			return fmt.Errorf("record %s: %w", m.Name, err)
		}
	}
	return nil
}
