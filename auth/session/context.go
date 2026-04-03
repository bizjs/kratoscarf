package session

import "context"

type sessionKey struct{}

// ContextWithSession returns a new context with the given session.
func ContextWithSession(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionKey{}, sess)
}

// FromContext extracts the Session from context.
// Returns nil if no session is present.
func FromContext(ctx context.Context) *Session {
	sess, _ := ctx.Value(sessionKey{}).(*Session)
	return sess
}
