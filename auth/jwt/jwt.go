package jwt

import (
	"context"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

const minSecretLength = 32

// Config holds JWT-specific configuration.
type Config struct {
	Secret        string        `json:"secret" yaml:"secret"`
	Issuer        string        `json:"issuer" yaml:"issuer"`
	AccessExpiry  time.Duration `json:"accessExpiry" yaml:"accessExpiry"`   // default: 2h
	RefreshExpiry time.Duration `json:"refreshExpiry" yaml:"refreshExpiry"` // default: 168h (7d)
	SigningMethod string        `json:"signingMethod" yaml:"signingMethod"` // default: HS256
}

// Authenticator is the JWT-based authenticator implementation.
type Authenticator struct {
	cfg           Config
	signingMethod gojwt.SigningMethod
	tokenStore    TokenStore
	claimsFactory func() Claims
}

// Option is a functional option for configuring JWT Authenticator.
type Option func(*Authenticator)

// WithTokenStore sets a token store for revocation support.
func WithTokenStore(store TokenStore) Option {
	return func(a *Authenticator) { a.tokenStore = store }
}

// WithSigningMethod sets the JWT signing method.
func WithSigningMethod(method gojwt.SigningMethod) Option {
	return func(a *Authenticator) { a.signingMethod = method }
}

// WithClaimsFactory sets a custom claims factory function.
func WithClaimsFactory(fn func() Claims) Option {
	return func(a *Authenticator) { a.claimsFactory = fn }
}

// New creates a new JWT Authenticator with the given configuration.
// Returns an error if the secret is shorter than 32 bytes.
func New(cfg Config, opts ...Option) (*Authenticator, error) {
	if len(cfg.Secret) < minSecretLength {
		return nil, fmt.Errorf("jwt: secret must be at least %d bytes, got %d", minSecretLength, len(cfg.Secret))
	}

	a := &Authenticator{
		cfg:           cfg,
		signingMethod: gojwt.SigningMethodHS256,
		claimsFactory: func() Claims { return Claims{} },
	}

	if cfg.SigningMethod != "" {
		if m := gojwt.GetSigningMethod(cfg.SigningMethod); m != nil {
			a.signingMethod = m
		}
	}

	if a.cfg.AccessExpiry == 0 {
		a.cfg.AccessExpiry = 2 * time.Hour
	}
	if a.cfg.RefreshExpiry == 0 {
		a.cfg.RefreshExpiry = 168 * time.Hour
	}

	for _, opt := range opts {
		opt(a)
	}
	return a, nil
}

func (a *Authenticator) GenerateTokenPair(ctx context.Context, claims Claims) (*TokenPair, error) {
	now := time.Now()

	claims.RegisteredClaims = gojwt.RegisteredClaims{
		Issuer:    a.cfg.Issuer,
		IssuedAt:  gojwt.NewNumericDate(now),
		ExpiresAt: gojwt.NewNumericDate(now.Add(a.cfg.AccessExpiry)),
	}

	accessToken := gojwt.NewWithClaims(a.signingMethod, claims)
	accessTokenString, err := accessToken.SignedString([]byte(a.cfg.Secret))
	if err != nil {
		return nil, err
	}

	refreshClaims := Claims{
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    a.cfg.Issuer,
			Subject:   claims.UserID,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(a.cfg.RefreshExpiry)),
		},
		UserID: claims.UserID,
	}

	refreshToken := gojwt.NewWithClaims(a.signingMethod, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(a.cfg.Secret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresAt:    now.Add(a.cfg.AccessExpiry),
	}, nil
}

func (a *Authenticator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims := a.claimsFactory()
	token, err := gojwt.ParseWithClaims(tokenString, &claims, func(token *gojwt.Token) (any, error) {
		if token.Method.Alg() != a.signingMethod.Alg() {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.cfg.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, gojwt.ErrTokenInvalidClaims
	}

	if a.tokenStore != nil {
		revoked, err := a.tokenStore.Exists(ctx, tokenString)
		if err != nil {
			return nil, err
		}
		if revoked {
			return nil, gojwt.ErrTokenInvalidClaims
		}
	}

	return &claims, nil
}

func (a *Authenticator) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := a.ValidateToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token to prevent replay.
	if err := a.RevokeToken(ctx, refreshToken); err != nil {
		return nil, err
	}

	return a.GenerateTokenPair(ctx, *claims)
}

func (a *Authenticator) RevokeToken(ctx context.Context, token string) error {
	if a.tokenStore == nil {
		return nil
	}
	return a.tokenStore.Store(ctx, token, a.cfg.AccessExpiry)
}
