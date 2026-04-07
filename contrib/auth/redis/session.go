package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/bizjs/kratoscarf/auth/session"
	goredis "github.com/redis/go-redis/v9"
)

// Compile-time interface check.
var _ session.Store = (*SessionStore)(nil)

// SessionStore implements session.Store using Redis.
type SessionStore struct {
	client    *goredis.Client
	keyPrefix string
}

// Option configures the SessionStore.
type Option func(*SessionStore)

// WithKeyPrefix sets the Redis key prefix (default: "session:").
func WithKeyPrefix(prefix string) Option {
	return func(s *SessionStore) { s.keyPrefix = prefix }
}

// NewSessionStore creates a new Redis-backed session store.
func NewSessionStore(client *goredis.Client, opts ...Option) *SessionStore {
	s := &SessionStore{
		client:    client,
		keyPrefix: "session:",
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *SessionStore) key(id string) string {
	return s.keyPrefix + id
}

func (s *SessionStore) Get(ctx context.Context, id string) (*session.Session, error) {
	data, err := s.client.Get(ctx, s.key(id)).Bytes()
	if err != nil {
		if err == goredis.Nil {
			return nil, nil
		}
		return nil, err
	}
	var sess session.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	if time.Now().After(sess.ExpiresAt) {
		_ = s.Delete(ctx, id)
		return nil, nil
	}
	return &sess, nil
}

func (s *SessionStore) Save(ctx context.Context, sess *session.Session) error {
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	ttl := time.Until(sess.ExpiresAt)
	if ttl <= 0 {
		return s.Delete(ctx, sess.ID)
	}
	return s.client.Set(ctx, s.key(sess.ID), data, ttl).Err()
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	return s.client.Del(ctx, s.key(id)).Err()
}
