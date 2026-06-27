// Package migrations_test contains lightweight structural tests for SQL migration files.
// These tests run without a live database — they only read and parse the SQL text.
package migrations_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// migrationsDir resolves the directory that contains the SQL migration files.
func migrationsDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(thisFile)
}

// readMigration reads a SQL file from the migrations directory.
func readMigration(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(migrationsDir(t), name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", name, err)
	}
	return string(data)
}

// TestSeed_ContainsAll12Slugs verifies R1: the up migration seeds exactly the
// 12 approved category slugs.
func TestSeed_ContainsAll12Slugs(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "000002_categories.up.sql")

	requiredSlugs := []string{
		"baches",
		"luminarias",
		"fuga-agua",
		"basura",
		"drenaje",
		"aguas-negras",
		"semaforo",
		"senaletica",
		"banquetas",
		"grafiti",
		"arboles",
		"animal-muerto",
	}

	for _, slug := range requiredSlugs {
		if !strings.Contains(sql, "'"+slug+"'") {
			t.Errorf("migration does not contain slug %q", slug)
		}
	}
}

// TestSeed_HasUniqueConstraintOnSlug verifies R6: the schema enforces slug uniqueness.
func TestSeed_HasUniqueConstraintOnSlug(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "000002_categories.up.sql")

	if !strings.Contains(strings.ToUpper(sql), "UNIQUE") {
		t.Error("migration must declare UNIQUE constraint on slug")
	}
}

// TestSeed_HasGenRandomUUID verifies R1/R6: UUIDs are auto-generated via gen_random_uuid().
func TestSeed_HasGenRandomUUID(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "000002_categories.up.sql")

	if !strings.Contains(sql, "gen_random_uuid()") {
		t.Error("migration must use gen_random_uuid() for the id default")
	}
}

// TestSeed_OnConflictDoNothing verifies idempotency: re-running the migration is safe.
func TestSeed_OnConflictDoNothing(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "000002_categories.up.sql")

	upper := strings.ToUpper(sql)
	if !strings.Contains(upper, "ON CONFLICT") || !strings.Contains(upper, "DO NOTHING") {
		t.Error("migration INSERT must be idempotent via ON CONFLICT (slug) DO NOTHING")
	}
}

// TestDown_DropsCategoriesTable verifies the rollback drops the table.
func TestDown_DropsCategoriesTable(t *testing.T) {
	t.Parallel()

	sql := readMigration(t, "000002_categories.down.sql")

	upper := strings.ToUpper(sql)
	if !strings.Contains(upper, "DROP TABLE") || !strings.Contains(upper, "CATEGORIES") {
		t.Error("down migration must DROP TABLE categories")
	}
}
