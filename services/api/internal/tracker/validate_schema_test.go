package tracker_test

import (
	"errors"
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
		"required field unknown": func(t *tracker.ItemType) { t.Transitions[2].RequiredFields = []tracker.FieldKey{"nope"} },
		"options on a text field": func(t *tracker.ItemType) {
			t.Fields[0].Options = []tracker.Option{{Key: "a", Name: "A"}}
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
	typ.Fields[0].Key = tracker.FieldStatus // "@status"

	err := (tracker.Schema{Types: []tracker.ItemType{typ}}).Validate()
	if !errors.Is(err, tracker.ErrReservedKey) {
		t.Errorf("Validate = %v, want ErrReservedKey", err)
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
