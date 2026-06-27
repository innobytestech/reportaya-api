package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Permission represents a granular permission (e.g. customers.create, portal.dashboard.read).
type Permission struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Code        string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"code"`
	Description string         `gorm:"type:text" json:"description"`
	Module      string         `gorm:"type:varchar(50)" json:"module"`
	CreatedAt   time.Time      `gorm:"column:created_at" json:"createdAt"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deletedAt"`
}

func (Permission) TableName() string { return "permissions" }
