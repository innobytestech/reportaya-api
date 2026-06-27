package migrate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	persistence "reportaya-api/internal/persistence"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Target defines one migration destination.
type Target struct {
	Name          string
	DatabaseURL   string
	MigrationsDir string
	Enabled       bool
}

// Run runs migrations from the given directory (e.g. "file://internal/persistence/migrations").
// databaseURL is the Postgres connection string. migrationsDir must be like "file://path/to/migrations".
// If env MIGRATE_FORCE_VERSION is set (e.g. "23"), Run forces that version first to clear a dirty state, then runs Up().
func Run(databaseURL, migrationsDir string) error {
	return RunWithOptions(databaseURL, migrationsDir, "default")
}

// RunTargets executes all enabled migration targets in order.
// Supports per-target forced versions with env MIGRATE_FORCE_VERSION_<TARGET_NAME>.
func RunTargets(targets []Target) error {
	for _, target := range targets {
		if !target.Enabled {
			continue
		}
		if strings.TrimSpace(target.DatabaseURL) == "" {
			return fmt.Errorf("migration target %q has empty database url", target.Name)
		}
		if err := RunWithOptions(target.DatabaseURL, target.MigrationsDir, target.Name); err != nil {
			return err
		}
	}
	return nil
}

// RunWithOptions runs migrations for one target with optional per-target force env var.
func RunWithOptions(databaseURL, migrationsDir, targetName string) error {
	if migrationsDir == "" {
		migrationsDir = "file://internal/persistence/migrations"
	}
	m, err := newMigrator(databaseURL, migrationsDir, targetName)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	if v := resolveForceVersion(targetName); v != "" {
		version, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("migrate force (%s): version must be a number, got %q", targetName, v)
		}
		if err := m.Force(version); err != nil {
			return fmt.Errorf("migrate force (%s, %d): %w", targetName, version, err)
		}
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up (%s): %w", targetName, err)
	}

	return nil
}

func newMigrator(databaseURL, migrationsDir, targetName string) (*migrate.Migrate, error) {
	normalizedDir, err := normalizeMigrationsDir(migrationsDir)
	if err == nil {
		m, err := migrate.New(normalizedDir, databaseURL)
		if err == nil {
			return m, nil
		}
		return nil, fmt.Errorf("migrate new (%s): %w", targetName, err)
	}

	subdir, ok := inferEmbeddedSubdir(migrationsDir)
	if !ok {
		return nil, fmt.Errorf("migrate dir (%s): %w", targetName, err)
	}

	src, embErr := iofs.New(persistence.EmbeddedMigrations, subdir)
	if embErr != nil {
		return nil, fmt.Errorf("migrate dir (%s): %w (embedded fallback failed: %w)", targetName, err, embErr)
	}

	m, embErr := migrate.NewWithSourceInstance("iofs", src, databaseURL)
	if embErr != nil {
		return nil, fmt.Errorf("migrate new (%s) with embedded fallback: %w", targetName, embErr)
	}

	return m, nil
}

func inferEmbeddedSubdir(migrationsDir string) (string, bool) {
	v := strings.ToLower(strings.TrimSpace(migrationsDir))
	switch {
	case strings.Contains(v, "networking-migrations"):
		return "networking-migrations", true
	case strings.Contains(v, "migrations"):
		return "migrations", true
	default:
		return "", false
	}
}

func resolveForceVersion(targetName string) string {
	normalized := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(targetName), "-", "_"))
	if normalized != "" {
		if v := os.Getenv("MIGRATE_FORCE_VERSION_" + normalized); v != "" {
			return v
		}
	}
	return os.Getenv("MIGRATE_FORCE_VERSION")
}

func normalizeMigrationsDir(migrationsDir string) (string, error) {
	if !strings.HasPrefix(migrationsDir, "file://") {
		return migrationsDir, nil
	}

	rawPath := strings.TrimPrefix(migrationsDir, "file://")
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", fmt.Errorf("empty file migration path")
	}

	path := filepath.Clean(filepath.FromSlash(rawPath))
	if filepath.IsAbs(path) {
		return toFileURI(path)
	}

	candidates, err := resolveRelativeCandidates(path)
	if err != nil {
		return "", err
	}
	for _, candidate := range candidates {
		if uri, err := toFileURI(candidate); err == nil {
			return uri, nil
		}
	}

	return "", fmt.Errorf("migrations directory %q not found from cwd/executable roots", rawPath)
}

func resolveRelativeCandidates(relPath string) ([]string, error) {
	out := make([]string, 0, 24)
	seen := make(map[string]struct{})
	appendCandidate := func(base string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		candidate := filepath.Clean(filepath.Join(base, relPath))
		if _, ok := seen[candidate]; ok {
			return
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}

	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}
	appendCandidate(wd)
	walkParents(wd, appendCandidate)

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		appendCandidate(exeDir)
		walkParents(exeDir, appendCandidate)
	}

	return out, nil
}

func walkParents(start string, fn func(string)) {
	cur := filepath.Clean(start)
	for i := 0; i < 10; i++ {
		parent := filepath.Dir(cur)
		if parent == cur {
			return
		}
		fn(parent)
		cur = parent
	}
}

func toFileURI(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("migrations path %q is not a directory", path)
	}
	abs := filepath.Clean(path)
	slash := filepath.ToSlash(abs)
	if vol := filepath.VolumeName(abs); vol != "" && !strings.HasPrefix(slash, "//") {
		return "file://" + slash, nil
	}
	if strings.HasPrefix(slash, "/") {
		return "file://" + slash, nil
	}
	return "file:///" + slash, nil
}
