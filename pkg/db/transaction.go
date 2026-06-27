package db

import "gorm.io/gorm"

// WithTransaction ejecuta fn dentro de una transacción.
// Si fn devuelve error, se hace rollback; si no, commit.
func WithTransaction(db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.Transaction(fn)
}
