package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Realm is staff or customer.
type Realm string

const (
	RealmStaff    Realm = "staff"
	RealmCustomer Realm = "customer"
)

// TokenType indicates access or refresh.
type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

// Claims holds JWT claims (sub=user_id, realm, customer_id, exp, iat).
//
// SessionStartedAt is only populated on refresh tokens and carries the unix epoch
// (seconds) of the original Login. It is propagated unchanged through every
// rotation so the AuthService can enforce an absolute session lifetime.
type Claims struct {
	jwt.RegisteredClaims
	Realm            Realm      `json:"realm"` // staff | customer
	CustomerID       *uuid.UUID `json:"customer_id,omitempty"`
	TokenType        TokenType  `json:"typ"`
	SessionStartedAt int64      `json:"sess_iat,omitempty"`
}

// UserID returns the subject as UUID (sub claim = user_id).
func (c *Claims) UserID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}
