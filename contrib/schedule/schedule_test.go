package schedule

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	s := NewScheduler()
	if s == nil {
		t.Fatal("NewScheduler returned nil")
	}
	if s.cron == nil {
		t.Fatal("cron instance is nil")
	}
	if s.jobs == nil {
		t.Fatal("jobs map is nil")
	}
	if s.running {
		t.Fatal("scheduler should not be running after creation")
	}
}

func TestAddJob(t *testing.T) {
	s := NewScheduler()

	err := s.AddJob(Job{
		Name:     "test-job",
		Schedule: "0 0 * * * *", // every hour
		Func:     func(ctx context.Context) error { return nil },
	})
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if _, ok := s.jobs["test-job"]; !ok {
		t.Fatal("job was not registered in the jobs map")
	}
}

func TestAddJob_DuplicateName(t *testing.T) {
	s := NewScheduler()

	job := Job{
		Name:     "dup",
		Schedule: "0 0 * * * *",
		Func:     func(ctx context.Context) error { return nil },
	}

	if err := s.AddJob(job); err != nil {
		t.Fatalf("first AddJob failed: %v", err)
	}

	err := s.AddJob(job)
	if err == nil {
		t.Fatal("expected error for duplicate job name, got nil")
	}
}

func TestAddJob_InvalidSchedule(t *testing.T) {
	s := NewScheduler()

	err := s.AddJob(Job{
		Name:     "bad-cron",
		Schedule: "not a cron expression",
		Func:     func(ctx context.Context) error { return nil },
	})
	if err == nil {
		t.Fatal("expected error for invalid cron expression, got nil")
	}
}

func TestAddInterval(t *testing.T) {
	s := NewScheduler()

	err := s.AddInterval("ticker", 5*time.Second, func(ctx context.Context) error { return nil })
	if err != nil {
		t.Fatalf("AddInterval failed: %v", err)
	}

	if _, ok := s.jobs["ticker"]; !ok {
		t.Fatal("interval job was not registered in the jobs map")
	}
}

func TestAddInterval_ZeroOrNegative(t *testing.T) {
	s := NewScheduler()

	err := s.AddInterval("zero", 0, func(ctx context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected error for zero interval, got nil")
	}

	err = s.AddInterval("neg", -1*time.Second, func(ctx context.Context) error { return nil })
	if err == nil {
		t.Fatal("expected error for negative interval, got nil")
	}
}

func TestAddInterval_DuplicateName(t *testing.T) {
	s := NewScheduler()

	fn := func(ctx context.Context) error { return nil }

	if err := s.AddInterval("dup", time.Second, fn); err != nil {
		t.Fatalf("first AddInterval failed: %v", err)
	}

	err := s.AddInterval("dup", time.Second, fn)
	if err == nil {
		t.Fatal("expected error for duplicate interval name, got nil")
	}
}

func TestRemoveJob(t *testing.T) {
	s := NewScheduler()

	if err := s.AddJob(Job{
		Name:     "removable",
		Schedule: "0 0 * * * *",
		Func:     func(ctx context.Context) error { return nil },
	}); err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	s.RemoveJob("removable")

	if _, ok := s.jobs["removable"]; ok {
		t.Fatal("job should have been removed from the jobs map")
	}
}

func TestRemoveJob_NonExistent(t *testing.T) {
	s := NewScheduler()

	// Should not panic or error.
	s.RemoveJob("does-not-exist")
}

func TestStartStop(t *testing.T) {
	s := NewScheduler()

	s.Start()
	if !s.running {
		t.Fatal("scheduler should be running after Start")
	}

	s.Stop()
	if s.running {
		t.Fatal("scheduler should not be running after Stop")
	}
}

func TestDoubleStart(t *testing.T) {
	s := NewScheduler()

	s.Start()
	s.Start() // second call should be safe
	if !s.running {
		t.Fatal("scheduler should still be running after double Start")
	}

	s.Stop()
}

func TestDoubleStop(t *testing.T) {
	s := NewScheduler()

	s.Start()
	s.Stop()
	s.Stop() // second call should be safe
	if s.running {
		t.Fatal("scheduler should not be running after double Stop")
	}
}

func TestStopWithoutStart(t *testing.T) {
	s := NewScheduler()

	// Should not panic.
	s.Stop()
	if s.running {
		t.Fatal("scheduler should not be running")
	}
}

func TestJobExecution(t *testing.T) {
	s := NewScheduler()

	var count atomic.Int64
	done := make(chan struct{}, 1)

	err := s.AddJob(Job{
		Name:     "counter",
		Schedule: "@every 1s",
		Func: func(ctx context.Context) error {
			if count.Add(1) == 1 {
				select {
				case done <- struct{}{}:
				default:
				}
			}
			return nil
		},
	})
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	s.Start()
	defer s.Stop()

	select {
	case <-done:
		// Job executed at least once.
	case <-time.After(3 * time.Second):
		t.Fatal("job did not execute within 3 seconds")
	}

	if v := count.Load(); v < 1 {
		t.Fatalf("expected count >= 1, got %d", v)
	}
}

func TestIntervalExecution(t *testing.T) {
	s := NewScheduler()

	var count atomic.Int64
	done := make(chan struct{}, 1)

	err := s.AddInterval("interval-counter", 1*time.Second, func(ctx context.Context) error {
		if count.Add(1) == 1 {
			select {
			case done <- struct{}{}:
			default:
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AddInterval failed: %v", err)
	}

	s.Start()
	defer s.Stop()

	select {
	case <-done:
		// Interval job executed at least once.
	case <-time.After(3 * time.Second):
		t.Fatal("interval job did not execute within 3 seconds")
	}

	if v := count.Load(); v < 1 {
		t.Fatalf("expected count >= 1, got %d", v)
	}
}

func TestRemoveJobStopsExecution(t *testing.T) {
	s := NewScheduler()

	var count atomic.Int64

	err := s.AddInterval("removable-ticker", 500*time.Millisecond, func(ctx context.Context) error {
		count.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("AddInterval failed: %v", err)
	}

	s.Start()
	defer s.Stop()

	// Wait for at least one execution.
	time.Sleep(1 * time.Second)

	s.RemoveJob("removable-ticker")
	snapshot := count.Load()

	// Wait and verify the count did not increase.
	time.Sleep(1 * time.Second)

	if count.Load() > snapshot+1 {
		// Allow at most 1 extra execution that may have been in-flight.
		t.Fatalf("job kept running after removal: count went from %d to %d", snapshot, count.Load())
	}
}
