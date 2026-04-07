package data

import (
	"context"
	"fmt"
	"sync"

	"github.com/bizjs/kratoscarf/examples/app/router-validation-demo/internal/biz"
	"github.com/bizjs/kratoscarf/response"
)

// todoRepo is an in-memory implementation of biz.TodoRepo.
type todoRepo struct {
	mu    sync.RWMutex
	todos map[string]*biz.Todo
	order []string // maintains insertion order
}

// NewTodoRepo creates an in-memory TodoRepo with pre-seeded data.
func NewTodoRepo() biz.TodoRepo {
	r := &todoRepo{
		todos: make(map[string]*biz.Todo),
		order: make([]string, 0),
	}
	// Seed some sample data.
	seeds := []*biz.Todo{
		{ID: "1", Title: "Learn kratoscarf router", Completed: true},
		{ID: "2", Title: "Add request validation", Completed: false},
		{ID: "3", Title: "Build a REST API", Completed: false},
	}
	for _, t := range seeds {
		r.todos[t.ID] = t
		r.order = append(r.order, t.ID)
	}
	return r
}

func (r *todoRepo) List(_ context.Context, req response.PageRequest) (*response.PageResponse[*biz.Todo], error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	req.Normalize(10, 100)

	all := make([]*biz.Todo, 0, len(r.order))
	for _, id := range r.order {
		if t, ok := r.todos[id]; ok {
			all = append(all, t)
		}
	}

	total := int64(len(all))
	start := req.Offset()
	if start > len(all) {
		start = len(all)
	}
	end := start + req.PageSize
	if end > len(all) {
		end = len(all)
	}

	return response.NewPageResponse(all[start:end], total, req), nil
}

func (r *todoRepo) Get(_ context.Context, id string) (*biz.Todo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.todos[id]
	if !ok {
		return nil, fmt.Errorf("todo %s not found", id)
	}
	return t, nil
}

func (r *todoRepo) Create(_ context.Context, todo *biz.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.todos[todo.ID]; exists {
		return fmt.Errorf("todo %s already exists", todo.ID)
	}
	r.todos[todo.ID] = todo
	r.order = append(r.order, todo.ID)
	return nil
}

func (r *todoRepo) Update(_ context.Context, todo *biz.Todo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.todos[todo.ID]; !ok {
		return fmt.Errorf("todo %s not found", todo.ID)
	}
	r.todos[todo.ID] = todo
	return nil
}

func (r *todoRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.todos[id]; !ok {
		return fmt.Errorf("todo %s not found", id)
	}
	delete(r.todos, id)
	newOrder := make([]string, 0, len(r.order)-1)
	for _, oid := range r.order {
		if oid != id {
			newOrder = append(newOrder, oid)
		}
	}
	r.order = newOrder
	return nil
}
