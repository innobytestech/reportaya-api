package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRealm is staff (internal) or customer (portal).
type UserRealm string

const (
	RealmStaff    UserRealm = "STAFF"
	RealmCustomer UserRealm = "CUSTOMER"
)

// User represents a unified user (staff or customer-scoped).
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string         `gorm:"type:varchar(50);not null"`
	Email        string         `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash string         `gorm:"type:varchar(255);not null;column:password_hash"`
	FullName     string         `gorm:"type:varchar(100);not null;column:full_name"`
	AvatarURL    *string        `gorm:"type:varchar(500);column:avatar_url"`
	Realm        UserRealm      `gorm:"type:user_realm;not null"`
	CustomerID   *uuid.UUID     `gorm:"type:uuid;column:customer_id"`
	IsActive     bool           `gorm:"default:true;column:is_active"`
	LastLoginAt  *time.Time     `gorm:"column:last_login_at"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"column:deleted_at;index"`
}

func (User) TableName() string { return "users" }
