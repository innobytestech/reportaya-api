package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// SignerVerifier signs and verifies JWTs.
type SignerVerifier struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

// NewSignerVerifier creates a JWT signer/verifier.
func NewSignerVerifier(secret string, expiration time.Duration, issuer string) *SignerVerifier {
	return &SignerVerifier{
		secret:     []byte(secret),
		expiration: expiration,
		issuer:     issuer,
	}
}

// Sign creates a signed access JWT for the given user/realm/customer.
func (s *SignerVerifier) Sign(userID uuid.UUID, realm Realm, customerID *uuid.UUID) (string, string, error) {
	return s.SignWithType(userID, realm, customerID, TokenTypeAccess)
}

// SignWithType creates a signed JWT with the given token type and returns token string and token ID (jti).
func (s *SignerVerifier) SignWithType(userID uuid.UUID, realm Realm, customerID *uuid.UUID, tokenType TokenType) (string, string, error) {
	return s.signClaims(userID, realm, customerID, tokenType, s.expiration, 0)
}

// SignRefreshWithSession creates a refresh token that carries SessionStartedAt
// and an explicit TTL. The TTL must be passed by the caller because the session
// horizon may shrink as the absolute timeout approaches.
func (s *SignerVerifier) SignRefreshWithSession(userID uuid.UUID, realm Realm, customerID *uuid.UUID, sessionStartedAt int64, ttl time.Duration) (string, string, error) {
	return s.signClaims(userID, realm, customerID, TokenTypeRefresh, ttl, sessionStartedAt)
}

func (s *SignerVerifier) signClaims(userID uuid.UUID, realm Realm, customerID *uuid.UUID, tokenType TokenType, ttl time.Duration, sessionStartedAt int64) (string, string, error) {
	now := time.Now().UTC()
	tokenID := uuid.Must(uuid.NewRandom()).String()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID.String(),
			Audience:  jwt.ClaimStrings{string(tokenType)},
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        tokenID,
		},
		Realm:            realm,
		CustomerID:       customerID,
		TokenType:        tokenType,
		SessionStartedAt: sessionStartedAt,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", "", err
	}
	return signed, tokenID, nil
}

// Verify parses and validates the token for the expected audience (token type), returns claims or error.
// It enforces: exact HS256 signing method, issuer match, and audience match.
func (s *SignerVerifier) Verify(tokenString string, expectedAudience TokenType) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	},
		jwt.WithIssuer(s.issuer),
		jwt.WithAudience(string(expectedAudience)),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token or claims")
	}
	return claims, nil
}
