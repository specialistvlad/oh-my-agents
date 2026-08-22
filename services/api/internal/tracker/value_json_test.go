package tracker_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Every kind must survive storage, or a store silently corrupts whatever it
// cannot express.
func TestEveryKindRoundTrips(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 30, 0, 0, time.UTC)
	for name, v := range map[string]tracker.Value{
		"text":         tracker.Text("hello"),
		"markdown":     tracker.Markdown("# hi"),
		"url":          tracker.URL("https://example.test"),
		"number":       tracker.Number(4.5),
		"bool":         tracker.Bool(true),
		"date":         tracker.Date(now),
		"duration":     tracker.Duration(90 * time.Second),
		"select":       tracker.Select("high-5n8p"),
		"multi-select": tracker.MultiSelect("ui-7q1s", "api-3t5u"),
		"actor":        tracker.Actor(tracker.ActorRef{Kind: tracker.ActorAgent, ID: "a1"}),
		"item":         tracker.ItemRef("i1"),
		"zero":         {},
	} {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(v)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var back tracker.Value
			if err := json.Unmarshal(encoded, &back); err != nil {
				t.Fatalf("Unmarshal %s: %v", encoded, err)
			}
			if back.Kind() != v.Kind() {
				t.Errorf("Kind = %q, want %q", back.Kind(), v.Kind())
			}
			if !back.Equal(v) {
				t.Errorf("round trip changed the value: %s", encoded)
			}
		})
	}
}

// The point of a filesystem store is that a person can read it, so the stored
// form has to be legible rather than an opaque encoding.
func TestStoredFormIsReadable(t *testing.T) {
	cases := map[string]struct {
		value tracker.Value
		want  string
	}{
		"date":     {tracker.Date(time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)), `"2026-08-21T12:00:00Z"`},
		"duration": {tracker.Duration(90 * time.Second), `"1m30s"`},
		"select":   {tracker.Select("high-5n8p"), `"high-5n8p"`},
		"number":   {tracker.Number(4.5), `4.5`},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			encoded, err := json.Marshal(c.value)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			if !strings.Contains(string(encoded), c.want) {
				t.Errorf("encoded as %s, want it to contain %s", encoded, c.want)
			}
			if !strings.Contains(string(encoded), string(c.value.Kind())) {
				t.Errorf("encoded as %s, want the kind named in it", encoded)
			}
		})
	}
}

// A file edited by hand into an impossible shape must fail to load rather
// than become a value whose kind lies about its payload.
func TestImpossibleStoredValuesAreRefused(t *testing.T) {
	for name, encoded := range map[string]string{
		"number holding a string": `{"kind":"number","value":"not a number"}`,
		"bool holding a number":   `{"kind":"bool","value":7}`,
		"date that is not a date": `{"kind":"date","value":"the third of never"}`,
		"duration that is not":    `{"kind":"duration","value":"a while"}`,
		"select holding a list":   `{"kind":"select","value":["a","b"]}`,
		"unknown kind":            `{"kind":"invented","value":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var v tracker.Value
			if err := json.Unmarshal([]byte(encoded), &v); err == nil {
				t.Errorf("accepted %s as %s", encoded, v.Kind())
			}
		})
	}
}

// An item carries a map of values, which is how they actually travel.
func TestValuesRoundTripInsideAnItem(t *testing.T) {
	item := openBug()
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back tracker.Item
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(back.Fields) != len(item.Fields) {
		t.Fatalf("Fields = %d, want %d", len(back.Fields), len(item.Fields))
	}
	for id, want := range item.Fields {
		if got := back.Fields[id]; !got.Equal(want) {
			t.Errorf("field %q changed: %v -> %v", id, want, got)
		}
	}
}
