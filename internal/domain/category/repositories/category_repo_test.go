package repositories_test

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"reportaya-api/internal/domain/category/repositories"
)

// newMockGormDB creates a GORM DB backed by go-sqlmock using regexp query matching.
func newMockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}

	dialector := postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })
	return db, mock
}

// TestListActive_FiltersIsActiveAndOrdersByName verifies R4 (is_active filter) and R5 (name ASC).
func TestListActive_FiltersIsActiveAndOrdersByName(t *testing.T) {
	t.Parallel()

	db, mock := newMockGormDB(t)

	cols := []string{"id", "slug", "name", "icon", "color", "is_active", "created_at", "updated_at", "deleted_at"}
	rows := sqlmock.NewRows(cols).
		AddRow("00000000-0000-0000-0000-000000000001", "baches", "Baches", "pothole", "#E11D48", true, time.Now(), time.Now(), nil).
		AddRow("00000000-0000-0000-0000-000000000002", "luminarias", "Luminarias apagadas", "lightbulb", "#F59E0B", true, time.Now(), time.Now(), nil)

	// GORM generates: SELECT * FROM "categories" WHERE is_active = $1 AND "categories"."deleted_at" IS NULL ORDER BY name ASC
	// Use a regexp that matches the essential filtering and ordering clauses.
	mock.ExpectQuery(`SELECT \* FROM "categories" WHERE is_active = \$1 AND "categories"\."deleted_at" IS NULL ORDER BY name ASC`).
		WillReturnRows(rows)

	repo := repositories.NewCategoryRepository(db)
	cats, err := repo.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive error: %v", err)
	}

	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	if cats[0].Slug != "baches" {
		t.Errorf("expected first slug 'baches', got %q", cats[0].Slug)
	}
	if cats[1].Slug != "luminarias" {
		t.Errorf("expected second slug 'luminarias', got %q", cats[1].Slug)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled mock expectations: %v", err)
	}
}
