package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

// Config holds session configuration.
type Config struct {
	MaxAge     time.Duration `json:"maxAge" yaml:"maxAge"`         // default: 24h
	CookieName string        `json:"cookieName" yaml:"cookieName"` // default: "session_id"
	CookiePath string        `json:"cookiePath" yaml:"cookiePath"` // default: "/"
	Domain     string        `json:"domain" yaml:"domain"`
	Secure     bool          `json:"secure" yaml:"secure"`     // default: false
	HTTPOnly   bool          `json:"httpOnly" yaml:"httpOnly"` // default: true
	SameSite   string        `json:"sameSite" yaml:"sameSite"` // "lax" (default) | "strict" | "none"
}

// Session represents a server-side session.
type Session struct {
	ID        string         `json:"id"`
	Values    map[string]any `json:"values"`
	CreatedAt time.Time      `json:"createdAt"`
	ExpiresAt time.Time      `json:"expiresAt"`
	IsNew     bool           `json:"-"`
	Modified  bool           `json:"-"`
}

// Set sets a value in the session.
func (s *Session) Set(key string, value any) {
	s.Values[key] = value
	s.Modified = true
}

// Get retrieves a value from the session.
func (s *Session) Get(key string) (any, bool) {
	v, ok := s.Values[key]
	return v, ok
}

// Delete removes a value from the session.
func (s *Session) Delete(key string) {
	delete(s.Values, key)
	s.Modified = true
}

// Store is the interface for session persistence.
type Store interface {
	Get(ctx context.Context, id string) (*Session, error)
	Save(ctx context.Context, session *Session) error
	Delete(ctx context.Context, id string) error
}

// Manager manages session lifecycle.
type Manager struct {
	store      Store
	maxAge     time.Duration
	cookieName string
	cookiePath string
	domain     string
	secure     bool
	httpOnly   bool
	sameSite   http.SameSite
	genID      func() string
}

// Option configures the Manager.
type Option func(*Manager)

// WithMaxAge sets the session max age.
func WithMaxAge(d time.Duration) Option { return func(m *Manager) { m.maxAge = d } }

// WithCookieName sets the session cookie name.
func WithCookieName(name string) Option { return func(m *Manager) { m.cookieName = name } }

// WithCookiePath sets the session cookie path.
func WithCookiePath(path string) Option { return func(m *Manager) { m.cookiePath = path } }

// WithCookieDomain sets the session cookie domain.
func WithCookieDomain(domain string) Option { return func(m *Manager) { m.domain = domain } }

// WithCookieSecure sets the session cookie Secure flag.
func WithCookieSecure(secure bool) Option { return func(m *Manager) { m.secure = secure } }

// WithCookieHTTPOnly sets the session cookie HttpOnly flag.
func WithCookieHTTPOnly(httpOnly bool) Option { return func(m *Manager) { m.httpOnly = httpOnly } }

// WithCookieSameSite sets the session cookie SameSite attribute.
func WithCookieSameSite(s http.SameSite) Option { return func(m *Manager) { m.sameSite = s } }

// WithIDGenerator sets a custom session ID generator function.
func WithIDGenerator(fn func() string) Option { return func(m *Manager) { m.genID = fn } }

// NewManager creates a Manager.
func NewManager(store Store, cfg Config, opts ...Option) *Manager {
	sameSite := http.SameSiteLaxMode
	switch cfg.SameSite {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	m := &Manager{
		store:      store,
		maxAge:     cfg.MaxAge,
		cookieName: cfg.CookieName,
		cookiePath: cfg.CookiePath,
		domain:     cfg.Domain,
		secure:     cfg.Secure,
		httpOnly:   cfg.HTTPOnly,
		sameSite:   sameSite,
		genID:      generateID,
	}

	if m.maxAge == 0 {
		m.maxAge = 24 * time.Hour
	}
	if m.cookieName == "" {
		m.cookieName = "session_id"
	}
	if m.cookiePath == "" {
		m.cookiePath = "/"
	}
	if !m.httpOnly {
		m.httpOnly = true
	}

	for _, opt := range opts {
		opt(m)
	}
	return m
}

// GetSession retrieves an existing session or creates a new one.
func (m *Manager) GetSession(ctx context.Context, r *http.Request) (*Session, error) {
	cookie, err := r.Cookie(m.cookieName)
	if err == nil && cookie.Value != "" {
		sess, err := m.store.Get(ctx, cookie.Value)
		if err != nil {
			return nil, err
		}
		if sess != nil {
			sess.IsNew = false
			sess.Modified = false
			return sess, nil
		}
	}

	now := time.Now()
	return &Session{
		ID:        m.genID(),
		Values:    make(map[string]any),
		CreatedAt: now,
		ExpiresAt: now.Add(m.maxAge),
		IsNew:     true,
		Modified:  true,
	}, nil
}

// SaveSession saves the session and sets the response cookie.
func (m *Manager) SaveSession(ctx context.Context, w http.ResponseWriter, sess *Session) error {
	if err := m.store.Save(ctx, sess); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    sess.ID,
		Path:     m.cookiePath,
		Domain:   m.domain,
		MaxAge:   int(m.maxAge.Seconds()),
		Secure:   m.secure,
		HttpOnly: m.httpOnly,
		SameSite: m.sameSite,
	})
	return nil
}

// DestroySession removes the session and clears the cookie.
func (m *Manager) DestroySession(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(m.cookieName)
	if err == nil && cookie.Value != "" {
		if err := m.store.Delete(ctx, cookie.Value); err != nil {
			return err
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     m.cookiePath,
		Domain:   m.domain,
		MaxAge:   -1,
		Secure:   m.secure,
		HttpOnly: m.httpOnly,
		SameSite: m.sameSite,
	})
	return nil
}

func generateID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("session: crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
