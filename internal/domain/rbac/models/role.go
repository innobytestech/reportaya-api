package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a role (staff or customer scope).
type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"type:varchar(50);not null" json:"name"`
	Description string         `gorm:"type:text" json:"description"`
	Realm       string         `gorm:"type:user_realm;not null" json:"realm"` // STAFF | CUSTOMER
	CreatedAt   time.Time      `gorm:"column:created_at" json:"createdAt"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index" json:"deletedAt"`

	Permissions []Permission `gorm:"many2many:role_permissions;" json:"permissions"`
}

func (Role) TableName() string { return "roles" }

// RolePermission links roles to permissions.
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID `gorm:"type:uuid;primaryKey;column:permission_id"`
}

func (RolePermission) TableName() string { return "role_permissions" }

// UserRole links users to roles.
type UserRole struct {
	UserID uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	RoleID uuid.UUID `gorm:"type:uuid;primaryKey;column:role_id"`
}

func (UserRole) TableName() string { return "user_roles" }
