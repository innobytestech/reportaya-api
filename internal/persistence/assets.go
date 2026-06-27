package persistence

import "embed"

// EmbeddedMigrations stores SQL migration files for runtime fallback when the
// migrations directory is not present next to the binary (e.g. distroless).
//
//go:embed migrations/*.sql
var EmbeddedMigrations embed.FS
