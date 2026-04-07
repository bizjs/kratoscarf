package schedule

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// Job represents a scheduled job.
type Job struct {
	Name     string // unique name for the job
	Schedule string // cron expression (e.g. "*/5 * * * *")
	Func     func(ctx context.Context) error
}

// Scheduler manages cron and interval jobs.
type Scheduler struct {
	cron    *cron.Cron
	jobs    map[string]cron.EntryID
	mu      sync.Mutex
	running bool
}

// NewScheduler creates a new Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		cron: cron.New(cron.WithSeconds()),
		jobs: make(map[string]cron.EntryID),
	}
}

// AddJob registers a cron job with a cron expression.
func (s *Scheduler) AddJob(job Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.checkName(job.Name); err != nil {
		return err
	}

	id, err := s.cron.AddFunc(job.Schedule, s.wrapFunc(job.Name, job.Func))
	if err != nil {
		return fmt.Errorf("schedule: failed to add job %q: %w", job.Name, err)
	}

	s.jobs[job.Name] = id
	return nil
}

// AddInterval registers a job that runs repeatedly at a fixed interval.
func (s *Scheduler) AddInterval(name string, interval time.Duration, fn func(ctx context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if interval <= 0 {
		return fmt.Errorf("schedule: interval must be positive, got %s", interval)
	}

	if err := s.checkName(name); err != nil {
		return err
	}

	id, err := s.cron.AddFunc(fmt.Sprintf("@every %s", interval), s.wrapFunc(name, fn))
	if err != nil {
		return fmt.Errorf("schedule: failed to add interval %q: %w", name, err)
	}

	s.jobs[name] = id
	return nil
}

// RemoveJob removes a job by name.
func (s *Scheduler) RemoveJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if id, ok := s.jobs[name]; ok {
		s.cron.Remove(id)
		delete(s.jobs, name)
	}
}

// Start starts the scheduler. It is non-blocking.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		s.cron.Start()
		s.running = true
	}
}

// Stop stops the scheduler gracefully, waiting for running jobs to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cron.Stop()
		s.running = false
	}
}

func (s *Scheduler) checkName(name string) error {
	if _, ok := s.jobs[name]; ok {
		return fmt.Errorf("schedule: job %q already registered", name)
	}
	return nil
}

func (s *Scheduler) wrapFunc(name string, fn func(ctx context.Context) error) func() {
	return func() {
		if err := fn(context.Background()); err != nil {
			log.Printf("schedule: job %q failed: %v", name, err)
		}
	}
}
