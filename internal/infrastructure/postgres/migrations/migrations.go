package migrations

import (
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed *.sql
var migrationFiles embed.FS

// Migration is one reversible schema change for the future shadow store.
type Migration struct {
	Name string
	Up   string
	Down string
}

// All returns embedded migrations in deterministic apply order.
func All() ([]Migration, error) {
	entries, err := migrationFiles.ReadDir(".")
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || strings.HasSuffix(name, ".down.sql") || !strings.HasSuffix(name, ".sql") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	migrations := make([]Migration, 0, len(names))
	for _, name := range names {
		up, err := readSQL(name)
		if err != nil {
			return nil, err
		}
		downName := strings.TrimSuffix(name, ".sql") + ".down.sql"
		down, err := readSQL(downName)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{Name: name, Up: up, Down: down})
	}
	return migrations, nil
}

func readSQL(name string) (string, error) {
	contents, err := migrationFiles.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", name, err)
	}
	return strings.TrimSpace(string(contents)), nil
}
