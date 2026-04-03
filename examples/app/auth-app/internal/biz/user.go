package biz

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrInvalidPassword = errors.New("invalid password")
)

// User represents a user in the system.
type User struct {
	ID       string
	Username string
	Password string // plaintext for demo
}

// UserRepo is the interface for user persistence.
type UserRepo interface {
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindByID(ctx context.Context, id string) (*User, error)
}

// UserUsecase handles user business logic.
type UserUsecase struct {
	repo UserRepo
}

func NewUserUsecase(repo UserRepo) *UserUsecase {
	return &UserUsecase{repo: repo}
}

// Authenticate validates credentials and returns the user.
func (uc *UserUsecase) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := uc.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	if user.Password != password {
		return nil, ErrInvalidPassword
	}
	return user, nil
}

// GetByID returns a user by ID.
func (uc *UserUsecase) GetByID(ctx context.Context, id string) (*User, error) {
	return uc.repo.FindByID(ctx, id)
}
