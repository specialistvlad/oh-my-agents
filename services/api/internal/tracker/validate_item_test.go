package tracker_test

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func TestValidateItem(t *testing.T) {
	s := bugSchema()
	if err := s.ValidateItem(openBug()); err != nil {
		t.Fatalf("ValidateItem: %v", err)
	}

	cases := map[string]struct {
		mutate func(*tracker.Item)
		want   error
	}{
		"unknown type":     {func(i *tracker.Item) { i.Type = "nope" }, tracker.ErrUnknownType},
		"unknown status":   {func(i *tracker.Item) { i.Status = "nope" }, tracker.ErrUnknownStatus},
		"undeclared field": {func(i *tracker.Item) { i.Fields["nope"] = tracker.Text("x") }, tracker.ErrUnknownField},
		"wrong kind":       {func(i *tracker.Item) { i.Fields["summary"] = tracker.Number(1) }, tracker.ErrKindMismatch},
		"missing required": {func(i *tracker.Item) { delete(i.Fields, "summary") }, tracker.ErrFieldRequired},
		"blank required":   {func(i *tracker.Item) { i.Fields["summary"] = tracker.Value{} }, tracker.ErrFieldRequired},
		"undeclared option": {
			func(i *tracker.Item) { i.Fields["severity"] = tracker.Select("catastrophic") },
			tracker.ErrUnknownOption,
		},
		"undeclared option in a multi-select": {
			func(i *tracker.Item) { i.Fields["tags"] = tracker.MultiSelect("ui", "invented") },
			tracker.ErrUnknownOption,
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			item := openBug()
			c.mutate(&item)
			if err := s.ValidateItem(item); !errors.Is(err, c.want) {
				t.Errorf("ValidateItem = %v, want %v", err, c.want)
			}
		})
	}
}

// A required field with a default is never actually missing, so creation has
// to judge required-ness after defaults rather than before.
func TestValidateNewCountsDefaultsAsPresent(t *testing.T) {
	typ := bugType()
	fallback := tracker.Text("unspecified")
	typ.Fields[0].Default = &fallback // summary is required
	s := tracker.Schema{Types: []tracker.ItemType{typ}}

	if err := s.ValidateNew(tracker.NewItem{Type: "bug", Title: "x"}); err != nil {
		t.Errorf("ValidateNew with a defaulted required field: %v", err)
	}

	// Without the default it is genuinely missing.
	if err := bugSchema().ValidateNew(tracker.NewItem{Type: "bug", Title: "x"}); !errors.Is(err, tracker.ErrFieldRequired) {
		t.Errorf("ValidateNew = %v, want ErrFieldRequired", err)
	}
}

func TestApplyDefaultsLeavesSuppliedValuesAlone(t *testing.T) {
	got := bugType().ApplyDefaults(map[tracker.FieldKey]tracker.Value{
		"severity": tracker.Select("high"),
	})
	if o, _ := got["severity"].Select(); o != "high" {
		t.Errorf("severity = %v, want the supplied value to win over the default", o)
	}
	if _, held := got["resolution"]; held {
		t.Error("a field with no default should stay absent")
	}
}
