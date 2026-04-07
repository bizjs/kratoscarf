package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckReadiness_AllUp(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("db", Readiness, func(ctx context.Context) error { return nil })
	r.RegisterFunc("redis", Readiness, func(ctx context.Context) error { return nil })

	report := r.CheckReadiness(context.Background())
	if report.Status != StatusUp {
		t.Fatalf("expected up, got %s", report.Status)
	}
	if len(report.Checks) != 2 {
		t.Fatalf("expected 2 checks, got %d", len(report.Checks))
	}
}

func TestCheckReadiness_OneDown(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("db", Readiness, func(ctx context.Context) error { return nil })
	r.RegisterFunc("redis", Readiness, func(ctx context.Context) error { return errors.New("connection refused") })

	report := r.CheckReadiness(context.Background())
	if report.Status != StatusDown {
		t.Fatalf("expected down, got %s", report.Status)
	}
	if report.Checks["redis"].Status != StatusDown {
		t.Fatal("expected redis down")
	}
	if report.Checks["redis"].Message != "connection refused" {
		t.Fatalf("unexpected message: %s", report.Checks["redis"].Message)
	}
}

func TestCheckLiveness_FiltersCorrectly(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("db", Readiness, func(ctx context.Context) error { return errors.New("down") })
	r.RegisterFunc("goroutine", Liveness, func(ctx context.Context) error { return nil })

	// Liveness should not include the readiness-only checker
	report := r.CheckLiveness(context.Background())
	if report.Status != StatusUp {
		t.Fatalf("expected up, got %s", report.Status)
	}
	if len(report.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(report.Checks))
	}
	if _, ok := report.Checks["goroutine"]; !ok {
		t.Fatal("expected goroutine check")
	}
}

func TestCheckType_Both(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("disk", Readiness|Liveness, func(ctx context.Context) error { return nil })

	liveness := r.CheckLiveness(context.Background())
	readiness := r.CheckReadiness(context.Background())

	if len(liveness.Checks) != 1 {
		t.Fatalf("expected 1 liveness check, got %d", len(liveness.Checks))
	}
	if len(readiness.Checks) != 1 {
		t.Fatalf("expected 1 readiness check, got %d", len(readiness.Checks))
	}
}

func TestNoCheckers_ReturnsUp(t *testing.T) {
	r := NewRegistry()
	report := r.CheckReadiness(context.Background())
	if report.Status != StatusUp {
		t.Fatalf("expected up, got %s", report.Status)
	}
}

func TestTimeout_RespectsContext(t *testing.T) {
	r := NewRegistry(WithTimeout(50 * time.Millisecond))
	r.RegisterFunc("slow", Readiness, func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			return nil
		}
	})

	report := r.CheckReadiness(context.Background())
	if report.Status != StatusDown {
		t.Fatalf("expected down due to timeout, got %s", report.Status)
	}
}

func TestConcurrentExecution(t *testing.T) {
	r := NewRegistry()
	for i := range 10 {
		name := string(rune('a' + i))
		r.RegisterFunc(name, Readiness, func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})
	}

	start := time.Now()
	report := r.CheckReadiness(context.Background())
	elapsed := time.Since(start)

	if report.Status != StatusUp {
		t.Fatalf("expected up, got %s", report.Status)
	}
	// 10 checkers at 50ms each — serial would be 500ms, concurrent should be ~50ms
	if elapsed > 300*time.Millisecond {
		t.Fatalf("expected concurrent execution, took %s", elapsed)
	}
}

func TestHandler_Healthy(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("db", Readiness, func(ctx context.Context) error { return nil })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	NewReadinessHandler(r).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandler_Unhealthy(t *testing.T) {
	r := NewRegistry()
	r.RegisterFunc("db", Readiness, func(ctx context.Context) error { return errors.New("down") })

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/readyz", nil)
	NewReadinessHandler(r).ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
