package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// PublicResourcePurpose identifies the resource kind a public-resource token
// grants access to (e.g. "quotation"). Keep stable — it's baked into audience.
type PublicResourcePurpose string

const (
	PublicResourceAudience = "public-resource"
)

// PublicResourceClaims are the claims for short-lived, unauthenticated,
// single-resource share tokens (e.g. a quotation view link sent to a client).
type PublicResourceClaims struct {
	jwt.RegisteredClaims
	Purpose PublicResourcePurpose `json:"purpose"`
}

// ResourceID returns the subject parsed as UUID.
func (c *PublicResourceClaims) ResourceID() (uuid.UUID, error) {
	return uuid.Parse(c.Subject)
}

// SignPublicResourceToken signs a token tied to a resource (by ID) with a
// custom expiration (instead of the global access/refresh expirations). Use
// for share links whose expiry should track the resource's own validity.
func (s *SignerVerifier) SignPublicResourceToken(resourceID uuid.UUID, purpose PublicResourcePurpose, expiresAt time.Time) (string, error) {
	now := time.Now().UTC()
	claims := &PublicResourceClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   resourceID.String(),
			Audience:  jwt.ClaimStrings{PublicResourceAudience},
			ExpiresAt: jwt.NewNumericDate(expiresAt.UTC()),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.Must(uuid.NewRandom()).String(),
		},
		Purpose: purpose,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", err
	}
	return signed, nil
}

// VerifyPublicResourceToken validates a public-resource token and returns its
// claims, or an error if expired/invalid/wrong purpose.
func (s *SignerVerifier) VerifyPublicResourceToken(tokenString string, expectedPurpose PublicResourcePurpose) (*PublicResourceClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PublicResourceClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(PublicResourceAudience),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*PublicResourceClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token or claims")
	}
	if claims.Purpose != expectedPurpose {
		return nil, fmt.Errorf("invalid token purpose")
	}
	return claims, nil
}
