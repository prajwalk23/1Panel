package github

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// SubmitResult says what happened to a submitted deploy request.
type SubmitResult int

const (
	// Started: nothing was running for the project, the request runs now.
	Started SubmitResult = iota
	// Queued: a run is in progress, the request will run right after it.
	Queued
	// Replaced: a run is in progress and a follow-up was already waiting;
	// the waiting one was dropped in favour of this newer request.
	Replaced
)

// Queue runs at most one job per project at a time and keeps at most one
// follow-up waiting. Pushes that arrive during a run collapse into that single
// follow-up, so the newest commit is always deployed last.
type Queue[T any] struct {
	mu   sync.Mutex
	jobs map[uint]*queueState[T]
	run  func(ctx context.Context, projectID uint, req T)
	wg   sync.WaitGroup
}

type queueState[T any] struct {
	running bool
	pending *T
	cancel  context.CancelFunc
}

// NewQueue creates a queue that calls run for every job. run must honour ctx
// cancellation.
func NewQueue[T any](run func(ctx context.Context, projectID uint, req T)) *Queue[T] {
	return &Queue[T]{jobs: map[uint]*queueState[T]{}, run: run}
}

// Submit starts req for projectID, or keeps it as the follow-up. When an older
// follow-up is replaced it is returned so the caller can mark it as skipped.
func (q *Queue[T]) Submit(projectID uint, req T) (SubmitResult, *T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	st := q.jobs[projectID]
	if st == nil {
		st = &queueState[T]{}
		q.jobs[projectID] = st
	}
	if st.running {
		dropped := st.pending
		st.pending = &req
		if dropped != nil {
			return Replaced, dropped
		}
		return Queued, nil
	}
	q.start(projectID, st, req)
	return Started, nil
}

// Cancel stops the running job of projectID and drops its follow-up. It
// returns whether a job was running, and the dropped follow-up if any.
func (q *Queue[T]) Cancel(projectID uint) (bool, *T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	st := q.jobs[projectID]
	if st == nil || !st.running {
		return false, nil
	}
	dropped := st.pending
	st.pending = nil
	if st.cancel != nil {
		st.cancel()
	}
	return true, dropped
}

// Running reports whether a job is in progress for projectID.
func (q *Queue[T]) Running(projectID uint) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	st := q.jobs[projectID]
	return st != nil && st.running
}

// Wait blocks until every started job has finished (used by tests and shutdown).
func (q *Queue[T]) Wait() {
	q.wg.Wait()
}

// start must be called with q.mu held.
func (q *Queue[T]) start(projectID uint, st *queueState[T], req T) {
	ctx, cancel := context.WithCancel(context.Background())
	st.running = true
	st.cancel = cancel
	q.wg.Add(1)
	go func() {
		defer q.wg.Done()
		defer cancel()
		q.run(ctx, projectID, req)

		q.mu.Lock()
		defer q.mu.Unlock()
		st.running = false
		st.cancel = nil
		if st.pending != nil {
			next := *st.pending
			st.pending = nil
			q.start(projectID, st, next)
		}
	}()
}

// Limiter is a per-project token bucket for webhook deliveries.
type Limiter struct {
	mu       sync.Mutex
	limiters map[uint]*rate.Limiter
	every    time.Duration
	burst    int
}

// NewLimiter allows burst deliveries at once, refilled one per every.
func NewLimiter(every time.Duration, burst int) *Limiter {
	return &Limiter{limiters: map[uint]*rate.Limiter{}, every: every, burst: burst}
}

// Allow reports whether a delivery for projectID may proceed now.
func (l *Limiter) Allow(projectID uint) bool {
	l.mu.Lock()
	lim := l.limiters[projectID]
	if lim == nil {
		lim = rate.NewLimiter(rate.Every(l.every), l.burst)
		l.limiters[projectID] = lim
	}
	l.mu.Unlock()
	return lim.Allow()
}
