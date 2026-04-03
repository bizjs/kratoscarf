package jwt

import (
	"net/http"
	"strings"
)

// bearerExtractor extracts tokens from the Authorization: Bearer <token> header.
type bearerExtractor struct{}

// BearerExtractor returns a TokenExtractor that extracts tokens from the
// Authorization: Bearer <token> header.
func BearerExtractor() TokenExtractor {
	return &bearerExtractor{}
}

// Extract extracts the bearer token from the request Authorization header.
func (e *bearerExtractor) Extract(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", ErrNoToken
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", ErrInvalidAuthHeader
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

// cookieExtractor extracts tokens from a named cookie.
type cookieExtractor struct {
	name string
}

// CookieExtractor returns a TokenExtractor that extracts tokens from a named cookie.
func CookieExtractor(name string) TokenExtractor {
	return &cookieExtractor{name: name}
}

// Extract extracts the token from the named cookie.
func (e *cookieExtractor) Extract(r *http.Request) (string, error) {
	cookie, err := r.Cookie(e.name)
	if err != nil {
		return "", ErrNoToken
	}
	if cookie.Value == "" {
		return "", ErrNoToken
	}
	return cookie.Value, nil
}

// queryExtractor extracts tokens from a query parameter.
type queryExtractor struct {
	param string
}

// QueryExtractor returns a TokenExtractor that extracts tokens from a query parameter.
func QueryExtractor(param string) TokenExtractor {
	return &queryExtractor{param: param}
}

// Extract extracts the token from the query parameter.
func (e *queryExtractor) Extract(r *http.Request) (string, error) {
	token := r.URL.Query().Get(e.param)
	if token == "" {
		return "", ErrNoToken
	}
	return token, nil
}

// chainExtractor tries multiple extractors in order, returning the first success.
type chainExtractor struct {
	extractors []TokenExtractor
}

// ChainExtractor returns a TokenExtractor that tries multiple extractors in order,
// returning the first successful extraction.
func ChainExtractor(extractors ...TokenExtractor) TokenExtractor {
	return &chainExtractor{extractors: extractors}
}

// Extract tries each extractor in order and returns the first successful token.
func (e *chainExtractor) Extract(r *http.Request) (string, error) {
	for _, ext := range e.extractors {
		token, err := ext.Extract(r)
		if err == nil {
			return token, nil
		}
	}
	return "", ErrNoToken
}
