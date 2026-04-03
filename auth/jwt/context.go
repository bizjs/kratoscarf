package jwt

import "context"

type claimsKey struct{}

// ContextWithClaims returns a new context with the given claims.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey{}, claims)
}

// ClaimsFromContext extracts JWT Claims from context.
// Returns nil if not authenticated via JWT.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsKey{}).(*Claims)
	return claims
}

// MustClaimsFromContext extracts JWT Claims from context.
// Panics if not authenticated via JWT.
func MustClaimsFromContext(ctx context.Context) *Claims {
	claims := ClaimsFromContext(ctx)
	if claims == nil {
		panic("auth/jwt: no claims in context")
	}
	return claims
}
