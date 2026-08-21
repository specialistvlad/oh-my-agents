package tracker_test

import (
	"testing"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func TestValueRoundTrips(t *testing.T) {
	now := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)

	if s, ok := tracker.Text("hi").String(); !ok || s != "hi" {
		t.Errorf("Text round trip = %q, %v", s, ok)
	}
	if n, ok := tracker.Number(4.5).Number(); !ok || n != 4.5 {
		t.Errorf("Number round trip = %v, %v", n, ok)
	}
	if b, ok := tracker.Bool(true).Bool(); !ok || !b {
		t.Errorf("Bool round trip = %v, %v", b, ok)
	}
	if d, ok := tracker.Date(now).Date(); !ok || !d.Equal(now) {
		t.Errorf("Date round trip = %v, %v", d, ok)
	}
	if d, ok := tracker.Duration(90 * time.Second).Duration(); !ok || d != 90*time.Second {
		t.Errorf("Duration round trip = %v, %v", d, ok)
	}
	if o, ok := tracker.Select("high").Select(); !ok || o != "high" {
		t.Errorf("Select round trip = %v, %v", o, ok)
	}
	if a, ok := tracker.Actor(agent("a1")).Actor(); !ok || a.ID != "a1" {
		t.Errorf("Actor round trip = %v, %v", a, ok)
	}
	if id, ok := tracker.ItemRef("i1").Item(); !ok || id != "i1" {
		t.Errorf("ItemRef round trip = %v, %v", id, ok)
	}
}

// The point of the typed accessors is that asking the wrong question fails
// instead of returning a plausible answer.
func TestValueAccessorsRefuseTheWrongKind(t *testing.T) {
	v := tracker.Number(1)
	if _, ok := v.String(); ok {
		t.Error("String() succeeded on a number")
	}
	if _, ok := v.Bool(); ok {
		t.Error("Bool() succeeded on a number")
	}
	if _, ok := v.Date(); ok {
		t.Error("Date() succeeded on a number")
	}
	if _, ok := v.Select(); ok {
		t.Error("Select() succeeded on a number")
	}
}

// Text, markdown and URL share a representation, and a caller that only
// wants to render text should not have to know which it holds.
func TestStringCoversEveryTextKind(t *testing.T) {
	for _, v := range []tracker.Value{
		tracker.Text("a"), tracker.Markdown("a"), tracker.URL("a"),
	} {
		if s, ok := v.String(); !ok || s != "a" {
			t.Errorf("String() on %s = %q, %v", v.Kind(), s, ok)
		}
	}
}

func TestZeroValueIsNotAnEmptyValue(t *testing.T) {
	var unset tracker.Value
	if !unset.IsZero() {
		t.Error("the zero Value should report IsZero")
	}
	if unset.Kind() != "" {
		t.Errorf("zero Kind = %q, want empty", unset.Kind())
	}
	if empty := tracker.Text(""); empty.IsZero() {
		t.Error("a field holding \"\" is not the same as an unset field")
	}
}

func TestMultiSelectCopiesBothWays(t *testing.T) {
	in := []tracker.OptionKey{"a", "b"}
	v := tracker.MultiSelect(in...)

	in[0] = "mutated"
	got, ok := v.MultiSelect()
	if !ok || got[0] != "a" {
		t.Errorf("value aliased the caller's slice: %v", got)
	}
	got[0] = "mutated"
	again, _ := v.MultiSelect()
	if again[0] != "a" {
		t.Errorf("value handed out its own slice: %v", again)
	}
}

func TestValueEquality(t *testing.T) {
	now := time.Now()
	cases := map[string]struct {
		a, b tracker.Value
		want bool
	}{
		"same text":         {tracker.Text("a"), tracker.Text("a"), true},
		"different text":    {tracker.Text("a"), tracker.Text("b"), false},
		"text vs markdown":  {tracker.Text("a"), tracker.Markdown("a"), false},
		"same instant":      {tracker.Date(now), tracker.Date(now.UTC()), true},
		"same options":      {tracker.MultiSelect("a", "b"), tracker.MultiSelect("a", "b"), true},
		"reordered options": {tracker.MultiSelect("a", "b"), tracker.MultiSelect("b", "a"), false},
		"two zero values":   {tracker.Value{}, tracker.Value{}, true},
		"zero vs empty":     {tracker.Value{}, tracker.Text(""), false},
	}
	for name, c := range cases {
		if got := c.a.Equal(c.b); got != c.want {
			t.Errorf("%s: Equal = %v, want %v", name, got, c.want)
		}
	}
}

func agent(id string) tracker.ActorRef {
	return tracker.ActorRef{Kind: tracker.ActorAgent, ID: id}
}
