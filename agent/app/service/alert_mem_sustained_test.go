package service

import (
	"testing"
	"time"
)

func TestMemSustainedStep(t *testing.T) {
	base := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	at := func(min int) time.Time { return base.Add(time.Duration(min) * time.Minute) }

	t.Run("fires after the full duration", func(t *testing.T) {
		var st memSustainedState
		for m := 0; m < 5; m++ {
			if memSustainedStep(at(m), 90, 80, 5, &st) {
				t.Fatalf("fired too early at minute %d", m)
			}
		}
		if !memSustainedStep(at(5), 90, 80, 5, &st) {
			t.Fatal("expected to fire at minute 5")
		}
	})

	t.Run("a dip below the threshold resets the timer", func(t *testing.T) {
		var st memSustainedState
		for m := 0; m < 4; m++ {
			memSustainedStep(at(m), 90, 80, 5, &st)
		}
		if memSustainedStep(at(4), 50, 80, 5, &st) {
			t.Fatal("must not fire below the threshold")
		}
		if memSustainedStep(at(5), 90, 80, 5, &st) {
			t.Fatal("timer must restart after a dip")
		}
		for m := 6; m < 10; m++ {
			if memSustainedStep(at(m), 90, 80, 5, &st) {
				t.Fatalf("fired too early at minute %d", m)
			}
		}
		if !memSustainedStep(at(10), 90, 80, 5, &st) {
			t.Fatal("expected to fire 5 minutes after the restart")
		}
	})

	t.Run("a sample gap re-arms the timer", func(t *testing.T) {
		var st memSustainedState
		memSustainedStep(at(0), 90, 80, 5, &st)
		memSustainedStep(at(1), 90, 80, 5, &st)
		if memSustainedStep(at(10), 90, 80, 5, &st) {
			t.Fatal("a 9 minute gap must not count as continuous")
		}
		for m := 11; m < 15; m++ {
			if memSustainedStep(at(m), 90, 80, 5, &st) {
				t.Fatalf("fired too early at minute %d", m)
			}
		}
		if !memSustainedStep(at(15), 90, 80, 5, &st) {
			t.Fatal("expected to fire 5 minutes after re-arming")
		}
	})

	t.Run("usage exactly at the threshold counts", func(t *testing.T) {
		var st memSustainedState
		memSustainedStep(at(0), 80, 80, 1, &st)
		if !memSustainedStep(at(1), 80, 80, 1, &st) {
			t.Fatal("usage equal to the threshold should fire")
		}
	})
}
