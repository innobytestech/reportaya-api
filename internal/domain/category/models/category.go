// Package models defines the persistence entities for the category domain.
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Category is a category of urban infrastructure report.
// It serves as the base entity for infrastructure issue classification (roads, utilities, signage, etc.).
// Fields marked as public are those exposed via the API contract (Slug, Name, Icon, Color);
// internal fields (ID, IsActive, timestamps) are excluded from public responses (R3).
type Category struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Slug      string         `gorm:"type:varchar(50);not null;uniqueIndex;column:slug"`
	Name      string         `gorm:"type:varchar(100);not null"`
	Icon      string         `gorm:"type:varchar(50);not null"`
	Color     string         `gorm:"type:varchar(7);not null"` // hex "#RRGGBB"
	IsActive  bool           `gorm:"default:true;column:is_active"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

// TableName returns the PostgreSQL table name for the Category entity.
// This method implements the gorm.Tabler interface to explicitly map the struct to the "categories" table.
func (Category) TableName() string { return "categories" }
