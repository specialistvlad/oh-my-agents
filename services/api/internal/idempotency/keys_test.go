package idempotency_test

import (
	"errors"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
)

type clock struct{ at time.Time }

func (c *clock) now() time.Time { return c.at }

func (c *clock) advance(d time.Duration) { c.at = c.at.Add(d) }

func newKeys(t *testing.T, ttl time.Duration, limit int) (*idempotency.Keys, *clock) {
	t.Helper()
	c := &clock{at: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return idempotency.New(idempotency.Options{TTL: ttl, Limit: limit, Now: c.now}), c
}

func TestRecallsWhatWasRemembered(t *testing.T) {
	keys, _ := newKeys(t, time.Minute, 10)
	wanted := errors.New("it failed")
	keys.Remember("k1", idempotency.Outcome{Err: wanted})

	got, ok := keys.Recall("k1")
	if !ok {
		t.Fatal("Recall found nothing for a key just remembered")
	}
	if !errors.Is(got.Err, wanted) {
		t.Errorf("Err = %v, want the original failure", got.Err)
	}
}

// A replay must be answered with the original outcome, failures included:
// reporting success for a command that failed is worse than duplicating it.
func TestRecallsSuccessAndFailureAlike(t *testing.T) {
	keys, _ := newKeys(t, time.Minute, 10)
	keys.Remember("ok", idempotency.Outcome{})

	got, ok := keys.Recall("ok")
	if !ok || got.Err != nil {
		t.Errorf("Recall = %+v, %v; want a remembered success", got, ok)
	}
}

func TestUnknownKeyIsNotRemembered(t *testing.T) {
	keys, _ := newKeys(t, time.Minute, 10)
	if _, ok := keys.Recall("never-seen"); ok {
		t.Error("Recall claimed to know a key it was never given")
	}
}

// An empty key means the caller wants the command to run every time.
func TestEmptyKeyIsNeverRemembered(t *testing.T) {
	keys, _ := newKeys(t, time.Minute, 10)
	keys.Remember("", idempotency.Outcome{})
	if _, ok := keys.Recall(""); ok {
		t.Error("an empty key was remembered")
	}
}

func TestForgetsAfterTheWindow(t *testing.T) {
	keys, c := newKeys(t, time.Minute, 10)
	keys.Remember("k", idempotency.Outcome{})

	c.advance(30 * time.Second)
	if _, ok := keys.Recall("k"); !ok {
		t.Error("forgot a key inside the window")
	}
	c.advance(2 * time.Minute)
	if _, ok := keys.Recall("k"); ok {
		t.Error("still remembers a key long past the window")
	}
}

// A client inventing keys must not be able to grow the map without bound.
func TestBoundedByTheLimit(t *testing.T) {
	keys, _ := newKeys(t, time.Hour, 8)
	for i := range 100 {
		keys.Remember(string(rune('a'+i%26))+string(rune('0'+i/26)), idempotency.Outcome{})
	}
	remembered := 0
	for i := range 100 {
		if _, ok := keys.Recall(string(rune('a'+i%26)) + string(rune('0'+i/26))); ok {
			remembered++
		}
	}
	if remembered > 8 {
		t.Errorf("holding %d keys, want at most the limit of 8", remembered)
	}
}

func TestSafeForConcurrentUse(t *testing.T) {
	keys, _ := newKeys(t, time.Minute, 1000)
	done := make(chan struct{})
	for worker := range 8 {
		go func() {
			defer func() { done <- struct{}{} }()
			for i := range 200 {
				key := string(rune('a'+worker)) + string(rune('0'+i%10))
				keys.Remember(key, idempotency.Outcome{})
				keys.Recall(key)
			}
		}()
	}
	for range 8 {
		<-done
	}
}
