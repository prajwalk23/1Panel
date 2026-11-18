package github

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestQueueOneRunningOneFollowUp(t *testing.T) {
	var mu sync.Mutex
	var ran []string
	release := make(chan struct{})
	started := make(chan string, 10)

	q := NewQueue(func(ctx context.Context, id uint, req string) {
		started <- req
		if req == "first" {
			<-release
		}
		mu.Lock()
		ran = append(ran, req)
		mu.Unlock()
	})

	if res, _ := q.Submit(1, "first"); res != Started {
		t.Fatalf("first: %v", res)
	}
	<-started
	if !q.Running(1) {
		t.Fatal("not running")
	}
	if res, dropped := q.Submit(1, "second"); res != Queued || dropped != nil {
		t.Fatalf("second: %v %v", res, dropped)
	}
	res, dropped := q.Submit(1, "third")
	if res != Replaced || dropped == nil || *dropped != "second" {
		t.Fatalf("third: %v %v", res, dropped)
	}
	// Another project is independent.
	if res, _ := q.Submit(2, "other"); res != Started {
		t.Fatalf("other project: %v", res)
	}

	close(release)
	q.Wait()

	mu.Lock()
	defer mu.Unlock()
	got := map[string]int{}
	for _, r := range ran {
		got[r]++
	}
	if got["first"] != 1 || got["third"] != 1 || got["other"] != 1 || got["second"] != 0 {
		t.Fatalf("ran %v", ran)
	}
	// "third" must run after "first".
	firstIdx, thirdIdx := -1, -1
	for i, r := range ran {
		if r == "first" {
			firstIdx = i
		}
		if r == "third" {
			thirdIdx = i
		}
	}
	if thirdIdx < firstIdx {
		t.Fatalf("order %v", ran)
	}
	if q.Running(1) {
		t.Fatal("still running")
	}
}

func TestQueueCancel(t *testing.T) {
	canceled := make(chan struct{})
	started := make(chan struct{}, 2)
	q := NewQueue(func(ctx context.Context, id uint, req int) {
		started <- struct{}{}
		select {
		case <-ctx.Done():
			close(canceled)
		case <-time.After(5 * time.Second):
		}
	})
	if ok, _ := q.Cancel(9); ok {
		t.Fatal("cancel of idle project returned true")
	}
	q.Submit(9, 1)
	<-started
	q.Submit(9, 2)
	ok, dropped := q.Cancel(9)
	if !ok || dropped == nil || *dropped != 2 {
		t.Fatalf("cancel: %v %v", ok, dropped)
	}
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("job did not see cancellation")
	}
	q.Wait()
	if len(started) != 0 {
		t.Fatal("dropped follow-up still ran")
	}
}

func TestLimiter(t *testing.T) {
	l := NewLimiter(time.Hour, 2)
	if !l.Allow(1) || !l.Allow(1) {
		t.Fatal("burst not allowed")
	}
	if l.Allow(1) {
		t.Fatal("limit not applied")
	}
	if !l.Allow(2) {
		t.Fatal("projects not independent")
	}
}
