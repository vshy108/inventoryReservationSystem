package migrations_test

import (
	"strings"
	"testing"

	"everest/inventoryReservation/internal/infrastructure/postgres/migrations"
)

func TestAll_ReturnsShadowPersistenceMigration(t *testing.T) {
	all, err := migrations.All()
	if err != nil {
		t.Fatalf("All returned error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 migration, got %d", len(all))
	}
	if all[0].Name != "0001_shadow_persistence.sql" {
		t.Fatalf("unexpected migration name: %s", all[0].Name)
	}
	if all[0].Up == "" || all[0].Down == "" {
		t.Fatal("migration must include both up and down SQL")
	}
}

func TestShadowPersistenceSchema_CoversPlannedTables(t *testing.T) {
	all, err := migrations.All()
	if err != nil {
		t.Fatalf("All returned error: %v", err)
	}
	schema := all[0].Up
	for _, fragment := range []string{
		"CREATE TABLE inventory_items",
		"CREATE TABLE reservations",
		"CREATE TABLE idempotency_keys",
		"CREATE TABLE outbox_events",
		"CHECK (total_stock >= confirmed_count + active_reservation_count)",
		"event_type TEXT NOT NULL CHECK",
		"observe_only BOOLEAN NOT NULL DEFAULT true",
	} {
		if !strings.Contains(schema, fragment) {
			t.Fatalf("schema missing fragment %q", fragment)
		}
	}
}

func TestShadowPersistenceRollback_DropsTablesInDependencyOrder(t *testing.T) {
	all, err := migrations.All()
	if err != nil {
		t.Fatalf("All returned error: %v", err)
	}
	rollback := all[0].Down
	expectedOrder := []string{
		"DROP TABLE IF EXISTS outbox_events",
		"DROP TABLE IF EXISTS idempotency_keys",
		"DROP TABLE IF EXISTS reservations",
		"DROP TABLE IF EXISTS inventory_items",
	}
	previous := -1
	for _, fragment := range expectedOrder {
		index := strings.Index(rollback, fragment)
		if index == -1 {
			t.Fatalf("rollback missing fragment %q", fragment)
		}
		if index <= previous {
			t.Fatalf("rollback fragment %q is out of dependency order", fragment)
		}
		previous = index
	}
}
