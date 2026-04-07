package middleware

import (
	"net/http"
	"strconv"

	kratoshttp "github.com/go-kratos/kratos/v2/transport/http"
)

// SecureConfig holds security header configuration.
type SecureConfig struct {
	// XContentTypeOptions sets X-Content-Type-Options. Default: "nosniff".
	XContentTypeOptions string
	// XFrameOptions sets X-Frame-Options. Default: "DENY".
	XFrameOptions string
	// ReferrerPolicy sets Referrer-Policy. Default: "strict-origin-when-cross-origin".
	ReferrerPolicy string
	// ContentSecurityPolicy sets Content-Security-Policy. Default: empty (not set).
	ContentSecurityPolicy string
	// HSTSMaxAge sets Strict-Transport-Security max-age in seconds.
	// Default: 0 (not set). Typical production value: 63072000 (2 years).
	HSTSMaxAge int
	// HSTSIncludeSubDomains adds includeSubDomains to HSTS. Default: false.
	HSTSIncludeSubDomains bool
}

// Secure returns an HTTP filter that sets security response headers.
// With no config, it applies safe defaults suitable for most applications.
//
//	kratoshttp.Filter(middleware.Secure(middleware.SecureConfig{}))
//	kratoshttp.Filter(middleware.Secure(middleware.SecureConfig{HSTSMaxAge: 63072000}))
func Secure(cfg SecureConfig) kratoshttp.FilterFunc {
	if cfg.XContentTypeOptions == "" {
		cfg.XContentTypeOptions = "nosniff"
	}
	if cfg.XFrameOptions == "" {
		cfg.XFrameOptions = "DENY"
	}
	if cfg.ReferrerPolicy == "" {
		cfg.ReferrerPolicy = "strict-origin-when-cross-origin"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", cfg.XContentTypeOptions)
			h.Set("X-Frame-Options", cfg.XFrameOptions)
			h.Set("Referrer-Policy", cfg.ReferrerPolicy)

			if cfg.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", cfg.ContentSecurityPolicy)
			}

			if cfg.HSTSMaxAge > 0 {
				var v string
				if cfg.HSTSIncludeSubDomains {
					v = "max-age=" + strconv.Itoa(cfg.HSTSMaxAge) + "; includeSubDomains"
				} else {
					v = "max-age=" + strconv.Itoa(cfg.HSTSMaxAge)
				}
				h.Set("Strict-Transport-Security", v)
			}

			next.ServeHTTP(w, r)
		})
	}
}
