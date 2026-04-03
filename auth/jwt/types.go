package jwt

import (
	"context"
	"errors"
	"net/http"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// Claims represents the standard JWT claims used by kratoscarf.
type Claims struct {
	gojwt.RegisteredClaims
	UserID   string            `json:"uid"`
	Username string            `json:"username,omitempty"`
	Roles    []string          `json:"roles,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

// TokenPair represents an access + refresh token pair.
type TokenPair struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

// TokenExtractor defines how to extract a token from a request.
type TokenExtractor interface {
	Extract(r *http.Request) (string, error)
}

// TokenStore provides persistent storage for tokens (used for revocation support).
type TokenStore interface {
	// Store saves a token with an expiration.
	Store(ctx context.Context, token string, expiration time.Duration) error
	// Exists checks if a token is stored (i.e., revoked).
	Exists(ctx context.Context, token string) (bool, error)
	// Delete removes a token from the store.
	Delete(ctx context.Context, token string) error
}

var (
	// ErrNoToken is returned when no token is found in the request.
	ErrNoToken = errors.New("auth: no token found")
	// ErrInvalidAuthHeader is returned when the Authorization header is malformed.
	ErrInvalidAuthHeader = errors.New("auth: invalid authorization header")
)
