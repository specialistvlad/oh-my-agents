package tracker_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func TestValidSchemaPasses(t *testing.T) {
	if err := bugSchema().Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

// Validating the root validates everything beneath it, so a fault buried in
// one field of one type surfaces from Schema.Validate.
func TestSchemaValidatesDownstream(t *testing.T) {
	s := bugSchema()
	s.Types[0].Fields[1].Options = nil // a select with no options

	err := s.Validate()
	if !errors.Is(err, tracker.ErrInvalidSchema) {
		t.Fatalf("Validate = %v, want ErrInvalidSchema from a nested field", err)
	}
}

func TestSchemaRejects(t *testing.T) {
	cases := map[string]func(*tracker.ItemType){
		"no statuses":            func(t *tracker.ItemType) { t.Statuses = nil },
		"initial not declared":   func(t *tracker.ItemType) { t.Initial = "nope" },
		"transition to nowhere":  func(t *tracker.ItemType) { t.Transitions[0].To = "nope" },
		"duplicate field":        func(t *tracker.ItemType) { t.Fields = append(t.Fields, t.Fields[0]) },
		"duplicate status":       func(t *tracker.ItemType) { t.Statuses = append(t.Statuses, t.Statuses[0]) },
		"duplicate transition":   func(t *tracker.ItemType) { t.Transitions = append(t.Transitions, t.Transitions[0]) },
		"unknown category":       func(t *tracker.ItemType) { t.Statuses[0].Category = "invented" },
		"unknown field kind":     func(t *tracker.ItemType) { t.Fields[0].Kind = "invented" },
		"required field unknown": func(t *tracker.ItemType) { t.Transitions[2].RequiredFields = []tracker.FieldID{"nope"} },
		"options on a text field": func(t *tracker.ItemType) {
			t.Fields[0].Options = []tracker.Option{{ID: "a", Name: "A"}}
		},
		"default of the wrong kind": func(t *tracker.ItemType) {
			wrong := tracker.Number(1)
			t.Fields[1].Default = &wrong
		},
		"default naming no option": func(t *tracker.ItemType) {
			wrong := tracker.Select("nonexistent")
			t.Fields[1].Default = &wrong
		},
	}
	for name, break_ := range cases {
		t.Run(name, func(t *testing.T) {
			typ := bugType()
			break_(&typ)
			if err := (tracker.Schema{Types: []tracker.ItemType{typ}}).Validate(); err == nil {
				t.Error("Validate accepted an inconsistent type")
			}
		})
	}
}

func TestSchemaRejectsDuplicateTypes(t *testing.T) {
	s := tracker.Schema{Types: []tracker.ItemType{bugType(), bugType()}}
	if err := s.Validate(); !errors.Is(err, tracker.ErrInvalidSchema) {
		t.Errorf("Validate = %v, want ErrInvalidSchema", err)
	}
}

func TestReservedFieldKeysAreRefused(t *testing.T) {
	typ := bugType()
	typ.Fields[0].ID = tracker.FieldStatus // "@status"

	err := (tracker.Schema{Types: []tracker.ItemType{typ}}).Validate()
	if !errors.Is(err, tracker.ErrReservedKey) {
		t.Errorf("Validate = %v, want ErrReservedKey", err)
	}
}

// Every level of the schema validates itself, Option included — it was the
// one that used to be checked inline instead.
func TestEveryLevelValidatesItself(t *testing.T) {
	levels := map[string]tracker.Validator{
		"schema":     bugSchema(),
		"item type":  bugType(),
		"field":      bugType().Fields[0],
		"status":     bugType().Statuses[0],
		"transition": bugType().Transitions[0],
		"option":     bugType().Fields[1].Options[0],
	}
	for name, level := range levels {
		if err := level.Validate(); err != nil {
			t.Errorf("%s: Validate = %v, want nil", name, err)
		}
	}
	if err := (tracker.Option{ID: "k"}).Validate(); !errors.Is(err, tracker.ErrInvalidSchema) {
		t.Errorf("Option with no name = %v, want ErrInvalidSchema", err)
	}
}

func TestCategoryResolved(t *testing.T) {
	resolved := map[tracker.StatusCategory]bool{
		tracker.CategoryBacklog:  false,
		tracker.CategoryActive:   false,
		tracker.CategoryBlocked:  false,
		tracker.CategoryDone:     true,
		tracker.CategoryCanceled: true,
	}
	for c, want := range resolved {
		if got := c.Resolved(); got != want {
			t.Errorf("%s.Resolved() = %v, want %v", c, got, want)
		}
	}
}

func TestMint(t *testing.T) {
	cases := map[string]struct{ name, want string }{
		"words":          {"In Review", "in-review-x"},
		"punctuation":    {"Won't Fix!", "won-t-fix-x"},
		"already a stem": {"bug", "bug-x"},
		"nothing usable": {"日本語", "x-x"},
		"very long": {
			"a very long status name that keeps going well past any sane limit",
			"a-very-long-status-name-that-keeps-going-x",
		},
	}
	for label, c := range cases {
		t.Run(label, func(t *testing.T) {
			if got := tracker.Mint(c.name, "x"); got != c.want {
				t.Errorf("Mint(%q) = %q, want %q", c.name, got, c.want)
			}
		})
	}
}

// Identifiers must look like something the system mints. A human-chosen key
// is exactly what ADR-0009 replaced, so the shapes people reach for first are
// the ones worth refusing.
func TestSchemaRefusesIdentifiersItCouldNotHaveMinted(t *testing.T) {
	cases := map[string]func(*tracker.ItemType){
		"capitals in a type":   func(t *tracker.ItemType) { t.ID = "Bug" },
		"underscore in a type": func(t *tracker.ItemType) { t.ID = "bug_9c2x" },
		"space in a status":    func(t *tracker.ItemType) { t.Statuses[0].ID = "in review" },
		"empty field id":       func(t *tracker.ItemType) { t.Fields[0].ID = "" },
		"dot in an option":     func(t *tracker.ItemType) { t.Fields[1].Options[0].ID = "low.4j6k" },
		"trailing hyphen":      func(t *tracker.ItemType) { t.ID = "bug-" },
		"path separator":       func(t *tracker.ItemType) { t.ID = "bug/9c2x" },
	}
	for name, break_ := range cases {
		t.Run(name, func(t *testing.T) {
			typ := bugType()
			break_(&typ)
			err := (tracker.Schema{Types: []tracker.ItemType{typ}}).Validate()
			if !errors.Is(err, tracker.ErrInvalidSchema) {
				t.Errorf("Validate = %v, want ErrInvalidSchema", err)
			}
		})
	}
}

// A minted id is usable everywhere it has to be: as a path segment and as a
// URL segment, which is what the grammar exists to guarantee.
func TestMintedIdentifiersAreSafeAsSegments(t *testing.T) {
	for _, name := range []string{"In Review", "Won't Fix!", "../escape", "a/b", "日本語"} {
		id := tracker.Mint(name, "4f7k")
		if strings.ContainsAny(id, `/\ .:?#%`) {
			t.Errorf("Mint(%q) = %q, which is not safe as a path or URL segment", name, id)
		}
		typ := bugType()
		typ.ID = tracker.TypeID(id)
		if err := (tracker.Schema{Types: []tracker.ItemType{typ}}).Validate(); err != nil {
			t.Errorf("Mint(%q) produced %q, which the schema rejects: %v", name, id, err)
		}
	}
}
