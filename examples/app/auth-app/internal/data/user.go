package data

import (
	"context"
	"sync"

	"github.com/bizjs/kratoscarf/examples/app/auth-app/internal/biz"
)

type userRepo struct {
	mu    sync.RWMutex
	users map[string]*biz.User // keyed by username
	byID  map[string]*biz.User // keyed by ID
}

// NewUserRepo creates a new in-memory user repository with demo users.
func NewUserRepo() biz.UserRepo {
	r := &userRepo{
		users: make(map[string]*biz.User),
		byID:  make(map[string]*biz.User),
	}
	// Pre-seed demo users
	for _, u := range []*biz.User{
		{ID: "1", Username: "alice", Password: "123456"},
		{ID: "2", Username: "bob", Password: "654321"},
	} {
		r.users[u.Username] = u
		r.byID[u.ID] = u
	}
	return r
}

func (r *userRepo) FindByUsername(_ context.Context, username string) (*biz.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[username]
	if !ok {
		return nil, biz.ErrUserNotFound
	}
	return u, nil
}

func (r *userRepo) FindByID(_ context.Context, id string) (*biz.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return nil, biz.ErrUserNotFound
	}
	return u, nil
}

func (r *userRepo) Create(_ context.Context, user *biz.User) (*biz.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.Username] = user
	r.byID[user.ID] = user
	return user, nil
}
