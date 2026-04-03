package conf

import "time"

// Bootstrap is the root configuration loaded from config.yaml.
type Bootstrap struct {
	Server ServerConfig `json:"server"`
	Auth   AuthConfig   `json:"auth"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	HTTP HTTPConfig `json:"http"`
}

// HTTPConfig holds HTTP listener configuration.
type HTTPConfig struct {
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Timeout string `json:"timeout"`
}

func (c *HTTPConfig) ParseTimeout() time.Duration {
	d, _ := time.ParseDuration(c.Timeout)
	return d
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWT     JWTConfig     `json:"jwt"`
	Session SessionConfig `json:"session"`
}

// JWTConfig holds JWT configuration.
type JWTConfig struct {
	Secret        string `json:"secret"`
	Issuer        string `json:"issuer"`
	AccessExpiry  string `json:"accessExpiry"`
	RefreshExpiry string `json:"refreshExpiry"`
}

// SessionConfig holds session configuration.
type SessionConfig struct {
	MaxAge     string `json:"maxAge"`
	CookieName string `json:"cookieName"`
}

func (c *JWTConfig) ParseAccessExpiry() time.Duration {
	d, _ := time.ParseDuration(c.AccessExpiry)
	if d == 0 {
		d = 2 * time.Hour
	}
	return d
}

func (c *JWTConfig) ParseRefreshExpiry() time.Duration {
	d, _ := time.ParseDuration(c.RefreshExpiry)
	if d == 0 {
		d = 168 * time.Hour
	}
	return d
}

func (c *SessionConfig) ParseMaxAge() time.Duration {
	d, _ := time.ParseDuration(c.MaxAge)
	if d == 0 {
		d = 24 * time.Hour
	}
	return d
}
