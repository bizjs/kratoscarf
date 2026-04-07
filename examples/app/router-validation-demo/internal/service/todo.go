package service

import (
	"context"
	"fmt"
	"strconv"

	kratosmiddleware "github.com/go-kratos/kratos/v2/middleware"

	"github.com/bizjs/kratoscarf/auth/session"
	"github.com/bizjs/kratoscarf/examples/app/router-validation-demo/internal/biz"
	"github.com/bizjs/kratoscarf/response"
	"github.com/bizjs/kratoscarf/router"
)

// Request types — validation rules are in struct tags.
// ctx.Bind() auto-validates because the router has a validator set.

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
}

type CreateTodoRequest struct {
	Title string `json:"title" validate:"required,min=1,max=200"`
}

type UpdateTodoRequest struct {
	Title     *string `json:"title" validate:"omitempty,min=1,max=200"`
	Completed *bool   `json:"completed"`
}

// TodoService handles HTTP requests for TODO resources.
type TodoService struct {
	uc      *biz.TodoBiz
	sessMgr *session.Manager
}

// NewTodoService creates a new TodoService.
func NewTodoService(uc *biz.TodoBiz, sessMgr *session.Manager) *TodoService {
	return &TodoService{uc: uc, sessMgr: sessMgr}
}

// RegisterRoutes registers all routes on the given router.
func (s *TodoService) RegisterRoutes(r *router.Router) {
	r.Use(session.Middleware(s.sessMgr))

	r.POST("/login", s.Login)
	r.POST("/logout", s.Logout)

	api := r.Group("/api/todos", requireSession())
	api.GET("", s.List)
	api.GET("/{id}", s.GetByID)
	api.POST("", s.Create)
	api.PUT("/{id}", s.Update)
	api.DELETE("/{id}", s.Delete)
}

// Login — ctx.Bind auto-validates LoginRequest.
func (s *TodoService) Login(ctx *router.Context) error {
	var req LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}

	sess := session.FromContext(ctx.Context())
	if sess == nil {
		return response.ErrInternal.WithMessage("session not available")
	}
	sess.Set("username", req.Username)
	if err := s.sessMgr.SaveSession(ctx.Context(), ctx.Response(), sess); err != nil {
		return response.ErrInternal.WithCause(err)
	}
	return ctx.Success(map[string]string{
		"message":  "logged in",
		"username": req.Username,
	})
}

// Logout destroys the session.
func (s *TodoService) Logout(ctx *router.Context) error {
	if err := s.sessMgr.DestroySession(ctx.Context(), ctx.Response(), ctx.Request()); err != nil {
		return response.ErrInternal.WithCause(err)
	}
	return ctx.Success(map[string]string{"message": "logged out"})
}

// List returns paginated TODOs.
func (s *TodoService) List(ctx *router.Context) error {
	page, _ := strconv.Atoi(ctx.QueryDefault("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.QueryDefault("pageSize", "10"))
	result, err := s.uc.List(ctx.Context(), response.PageRequest{Page: page, PageSize: pageSize})
	if err != nil {
		return response.ErrInternal.WithCause(err)
	}
	return ctx.Success(result)
}

// GetByID returns a single TODO.
func (s *TodoService) GetByID(ctx *router.Context) error {
	id := ctx.Param("id")
	todo, err := s.uc.Get(ctx.Context(), id)
	if err != nil {
		return response.ErrNotFound.WithMessage(fmt.Sprintf("todo %s not found", id))
	}
	return ctx.Success(todo)
}

// Create — ctx.Bind auto-validates CreateTodoRequest.
func (s *TodoService) Create(ctx *router.Context) error {
	var req CreateTodoRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}

	all, _ := s.uc.List(ctx.Context(), response.PageRequest{Page: 1, PageSize: 10000})
	newID := strconv.Itoa(int(all.Total) + 1)
	todo := &biz.Todo{ID: newID, Title: req.Title}
	if err := s.uc.Create(ctx.Context(), todo); err != nil {
		return response.ErrInternal.WithCause(err)
	}
	return ctx.JSON(201, response.Success(todo))
}

// Update — ctx.Bind auto-validates UpdateTodoRequest.
func (s *TodoService) Update(ctx *router.Context) error {
	id := ctx.Param("id")
	existing, err := s.uc.Get(ctx.Context(), id)
	if err != nil {
		return response.ErrNotFound.WithMessage(fmt.Sprintf("todo %s not found", id))
	}

	var req UpdateTodoRequest
	if err := ctx.Bind(&req); err != nil {
		return err
	}
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Completed != nil {
		existing.Completed = *req.Completed
	}
	if err := s.uc.Update(ctx.Context(), existing); err != nil {
		return response.ErrInternal.WithCause(err)
	}
	return ctx.Success(existing)
}

// Delete removes a TODO.
func (s *TodoService) Delete(ctx *router.Context) error {
	id := ctx.Param("id")
	if err := s.uc.Delete(ctx.Context(), id); err != nil {
		return response.ErrNotFound.WithMessage(fmt.Sprintf("todo %s not found", id))
	}
	return ctx.NoContent()
}

// requireSession blocks requests without an active session.
func requireSession() kratosmiddleware.Middleware {
	return func(handler kratosmiddleware.Handler) kratosmiddleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			sess := session.FromContext(ctx)
			if sess == nil || sess.IsNew {
				return nil, response.ErrUnauthorized.WithMessage("login required")
			}
			return handler(ctx, req)
		}
	}
}
