// Package settingstest is the conformance suite for [settings.Store].
//
// It exists because ADR-0005 puts enforcement in the adapter: the filesystem
// store checks keys and documents in Go, and a future SQL store would push
// the same rules into constraints. Rules implemented twice are only
// replaceable if both implementations agree, and this suite is what says
// they do. An adapter that has not passed it is not finished.
package settingstest

import (
	"context"
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// Factory builds a fresh, empty store for one subtest.
type Factory func(t *testing.T) settings.Store

// Run exercises every guarantee [settings.Store] documents.
func Run(t *testing.T, newStore Factory) {
	t.Helper()
	for name, test := range map[string]func(*testing.T, Factory){
		"round trip":            testRoundTrip,
		"overwrite":             testOverwrite,
		"missing key":           testMissing,
		"delete":                testDelete,
		"list is sorted":        testKeysSorted,
		"list is empty at rest": testKeysEmpty,
		"nested keys":           testNested,
		"rejects bad keys":      testBadKeys,
		"rejects bad documents": testBadDocuments,
		"returns copies":        testCopies,
	} {
		t.Run(name, func(t *testing.T) { test(t, newStore) })
	}
}

func testRoundTrip(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	want := settings.Document(`{"model":"opus"}`)
	if err := s.Set(ctx, "agent/model", want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := s.Get(ctx, "agent/model")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("Get = %s, want %s", got, want)
	}
}

func testOverwrite(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	mustSet(t, s, "k", `{"v":1}`)
	mustSet(t, s, "k", `{"v":2}`)
	got, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != `{"v":2}` {
		t.Errorf("Get = %s, want the second write", got)
	}
}

func testMissing(t *testing.T, newStore Factory) {
	s := newStore(t)
	if _, err := s.Get(t.Context(), "nope"); !errors.Is(err, settings.ErrNotFound) {
		t.Errorf("Get of absent key = %v, want ErrNotFound", err)
	}
}

func testDelete(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	mustSet(t, s, "k", `{}`)
	if err := s.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := s.Get(ctx, "k"); !errors.Is(err, settings.ErrNotFound) {
		t.Errorf("Get after Delete = %v, want ErrNotFound", err)
	}
	if err := s.Delete(ctx, "k"); !errors.Is(err, settings.ErrNotFound) {
		t.Errorf("second Delete = %v, want ErrNotFound", err)
	}
}

func testKeysSorted(t *testing.T, newStore Factory) {
	s := newStore(t)
	for _, k := range []string{"z", "a/b", "a/a", "m"} {
		mustSet(t, s, settings.Key(k), `{}`)
	}
	got, err := s.Keys(t.Context())
	if err != nil {
		t.Fatalf("Keys: %v", err)
	}
	want := []settings.Key{"a/a", "a/b", "m", "z"}
	if len(got) != len(want) {
		t.Fatalf("Keys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Keys = %v, want %v", got, want)
		}
	}
}

func testKeysEmpty(t *testing.T, newStore Factory) {
	got, err := newStore(t).Keys(t.Context())
	if err != nil {
		t.Fatalf("Keys on an untouched store: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Keys = %v, want none", got)
	}
}

func testNested(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	mustSet(t, s, "a/b/c/deep", `{"ok":true}`)
	if _, err := s.Get(ctx, "a/b/c/deep"); err != nil {
		t.Errorf("Get of a nested key: %v", err)
	}
}

func testBadKeys(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	for _, k := range []settings.Key{
		"", "..", "../escape", "a/../../etc/passwd", "/absolute",
		"trailing/", "double//slash", ".hidden", "sp ace", "bad*char",
	} {
		if err := s.Set(ctx, k, settings.Document("{}")); !errors.Is(err, settings.ErrInvalidKey) {
			t.Errorf("Set(%q) = %v, want ErrInvalidKey", k, err)
		}
		if _, err := s.Get(ctx, k); !errors.Is(err, settings.ErrInvalidKey) {
			t.Errorf("Get(%q) = %v, want ErrInvalidKey", k, err)
		}
		if err := s.Delete(ctx, k); !errors.Is(err, settings.ErrInvalidKey) {
			t.Errorf("Delete(%q) = %v, want ErrInvalidKey", k, err)
		}
	}
}

func testBadDocuments(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	for _, d := range []string{``, `not json`, `{"unclosed":`, `{}{}`} {
		doc := settings.Document(d)
		if err := s.Set(ctx, "k", doc); !errors.Is(err, settings.ErrInvalidDocument) {
			t.Errorf("Set(%q) = %v, want ErrInvalidDocument", d, err)
		}
	}
}

func testCopies(t *testing.T, newStore Factory) {
	s, ctx := newStore(t), t.Context()
	original := settings.Document(`{"v":1}`)
	if err := s.Set(ctx, "k", original); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Mutating what went in must not reach the store...
	original[5] = 'X'
	got, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != `{"v":1}` {
		t.Errorf("store kept the caller's slice: %s", got)
	}
	// ...nor must mutating what came out.
	got[5] = 'Y'
	again, err := s.Get(ctx, "k")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(again) != `{"v":1}` {
		t.Errorf("store handed out its own slice: %s", again)
	}
}

func mustSet(t *testing.T, s settings.Store, key settings.Key, doc string) {
	t.Helper()
	if err := s.Set(context.Background(), key, settings.Document(doc)); err != nil {
		t.Fatalf("Set(%q): %v", key, err)
	}
}
