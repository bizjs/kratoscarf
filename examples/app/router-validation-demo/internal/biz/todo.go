package biz

import (
	"context"

	"github.com/bizjs/kratoscarf/response"
)

// Todo represents a TODO item.
type Todo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// TodoRepo is the repository interface for TODO persistence.
type TodoRepo interface {
	List(ctx context.Context, req response.PageRequest) (*response.PageResponse[*Todo], error)
	Get(ctx context.Context, id string) (*Todo, error)
	Create(ctx context.Context, todo *Todo) error
	Update(ctx context.Context, todo *Todo) error
	Delete(ctx context.Context, id string) error
}

// TodoBiz orchestrates TODO business logic.
type TodoBiz struct {
	repo TodoRepo
}

// NewTodoBiz creates a new TodoBiz.
func NewTodoBiz(repo TodoRepo) *TodoBiz {
	return &TodoBiz{repo: repo}
}

// List returns paginated TODOs.
func (uc *TodoBiz) List(ctx context.Context, req response.PageRequest) (*response.PageResponse[*Todo], error) {
	return uc.repo.List(ctx, req)
}

// Get returns a single TODO by ID.
func (uc *TodoBiz) Get(ctx context.Context, id string) (*Todo, error) {
	return uc.repo.Get(ctx, id)
}

// Create adds a new TODO.
func (uc *TodoBiz) Create(ctx context.Context, todo *Todo) error {
	return uc.repo.Create(ctx, todo)
}

// Update modifies an existing TODO.
func (uc *TodoBiz) Update(ctx context.Context, todo *Todo) error {
	return uc.repo.Update(ctx, todo)
}

// Delete removes a TODO by ID.
func (uc *TodoBiz) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}
