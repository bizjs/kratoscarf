package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health status.
type Status string

const (
	StatusUp   Status = "up"
	StatusDown Status = "down"
)

// CheckType indicates which endpoints a checker participates in.
type CheckType int

const (
	Readiness CheckType = 1 << iota
	Liveness
)

// CheckResult is the result of a single health check.
type CheckResult struct {
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// Checker performs a single health check.
type Checker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// HealthReport is the aggregated result of all health checks.
type HealthReport struct {
	Status Status                 `json:"status"`
	Checks map[string]CheckResult `json:"checks,omitempty"`
}

// Option configures a Registry.
type Option func(*Registry)

// WithTimeout sets the per-check timeout. Default is 5s.
func WithTimeout(d time.Duration) Option {
	return func(r *Registry) { r.timeout = d }
}

type entry struct {
	checker Checker
	types   CheckType
}

// Registry holds all registered health checkers.
type Registry struct {
	mu      sync.RWMutex
	entries []entry
	timeout time.Duration
}

// NewRegistry creates a new health check registry.
func NewRegistry(opts ...Option) *Registry {
	r := &Registry{timeout: 5 * time.Second}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Register adds a checker for the given check types.
func (r *Registry) Register(checker Checker, types CheckType) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, entry{checker: checker, types: types})
}

// RegisterFunc is a shortcut for registering a simple check function.
func (r *Registry) RegisterFunc(name string, types CheckType, fn func(ctx context.Context) error) {
	r.Register(&funcChecker{name: name, fn: fn}, types)
}

// CheckLiveness runs checkers registered for liveness.
func (r *Registry) CheckLiveness(ctx context.Context) *HealthReport {
	return r.check(ctx, Liveness)
}

// CheckReadiness runs checkers registered for readiness.
func (r *Registry) CheckReadiness(ctx context.Context) *HealthReport {
	return r.check(ctx, Readiness)
}

func (r *Registry) check(ctx context.Context, typ CheckType) *HealthReport {
	r.mu.RLock()
	var matched []entry
	for _, e := range r.entries {
		if e.types&typ != 0 {
			matched = append(matched, e)
		}
	}
	r.mu.RUnlock()

	report := &HealthReport{
		Status: StatusUp,
		Checks: make(map[string]CheckResult, len(matched)),
	}
	if len(matched) == 0 {
		return report
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, e := range matched {
		wg.Add(1)
		go func(c Checker) {
			defer wg.Done()
			checkCtx, cancel := context.WithTimeout(ctx, r.timeout)
			defer cancel()
			result := c.Check(checkCtx)
			mu.Lock()
			report.Checks[c.Name()] = result
			if result.Status == StatusDown {
				report.Status = StatusDown
			}
			mu.Unlock()
		}(e.checker)
	}
	wg.Wait()

	return report
}

type funcChecker struct {
	name string
	fn   func(ctx context.Context) error
}

func (c *funcChecker) Name() string { return c.name }

func (c *funcChecker) Check(ctx context.Context) CheckResult {
	if err := c.fn(ctx); err != nil {
		return CheckResult{Status: StatusDown, Message: err.Error()}
	}
	return CheckResult{Status: StatusUp}
}
