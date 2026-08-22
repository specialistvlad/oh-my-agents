package trackertest

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func runSchema(t *testing.T, newStore Factory) {
	t.Run("round trips a type", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		schema, err := s.Schema(ctx)
		if err != nil {
			t.Fatalf("Schema: %v", err)
		}
		if _, ok := schema.Type(TypeBug); !ok {
			t.Error("the stored type is missing from the schema")
		}
	})

	t.Run("refuses an inconsistent type", func(t *testing.T) {
		s := newStore(t)
		broken := BugType()
		broken.Initial = "not-a-status"
		if err := s.PutItemType(t.Context(), broken); !errors.Is(err, tracker.ErrInvalidSchema) {
			t.Errorf("PutItemType = %v, want ErrInvalidSchema", err)
		}
	})

	// A schema that contradicts its own data is not a state worth reaching.
	t.Run("refuses a change that invalidates stored items", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		create(t, s, tracker.NewItem{})

		narrowed := BugType()
		narrowed.Statuses = narrowed.Statuses[1:] // drops StatusOpen, where items sit
		narrowed.Initial = StatusDoing
		narrowed.Transitions = nil
		if err := s.PutItemType(ctx, narrowed); !errors.Is(err, tracker.ErrInvalidSchema) {
			t.Errorf("PutItemType = %v, want ErrInvalidSchema", err)
		}
	})

	t.Run("refuses to delete a type in use", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		create(t, s, tracker.NewItem{})
		if err := s.DeleteItemType(ctx, TypeBug); err == nil {
			t.Error("DeleteItemType removed a type that still has items")
		}
	})

	t.Run("reports an unknown type", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		if err := s.DeleteItemType(ctx, "nope"); !errors.Is(err, tracker.ErrUnknownType) {
			t.Errorf("DeleteItemType = %v, want ErrUnknownType", err)
		}
	})
}

func runItems(t *testing.T, newStore Factory) {
	t.Run("creates at the initial status", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{Title: "It breaks"})
		if item.Status != StatusOpen {
			t.Errorf("Status = %q, want the type's initial status", item.Status)
		}
		if item.ID == "" {
			t.Error("CreateItem returned no ID")
		}
		if item.Version != 1 {
			t.Errorf("Version = %d, want 1", item.Version)
		}
	})

	t.Run("applies defaults", func(t *testing.T) {
		s, _ := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if o, _ := item.Fields[FieldSeverity].Select(); o != OptionMedium {
			t.Errorf("severity = %q, want the declared default", o)
		}
	})

	t.Run("enforces the schema on create", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		cases := map[string]struct {
			item tracker.NewItem
			want error
		}{
			"unknown type": {tracker.NewItem{Type: "nope"}, tracker.ErrUnknownType},
			"missing required": {
				tracker.NewItem{Type: TypeBug, Fields: map[tracker.FieldID]tracker.Value{}},
				tracker.ErrFieldRequired,
			},
			"undeclared field": {
				tracker.NewItem{Type: TypeBug, Fields: map[tracker.FieldID]tracker.Value{
					FieldSummary: tracker.Text("x"), "nope": tracker.Text("x"),
				}},
				tracker.ErrUnknownField,
			},
			"wrong kind": {
				tracker.NewItem{Type: TypeBug, Fields: map[tracker.FieldID]tracker.Value{
					FieldSummary: tracker.Number(1),
				}},
				tracker.ErrKindMismatch,
			},
			"undeclared option": {
				tracker.NewItem{Type: TypeBug, Fields: map[tracker.FieldID]tracker.Value{
					FieldSummary: tracker.Text("x"), FieldSeverity: tracker.Select("nope"),
				}},
				tracker.ErrUnknownOption,
			},
		}
		for name, c := range cases {
			t.Run(name, func(t *testing.T) {
				if _, err := s.CreateItem(ctx, c.item); !errors.Is(err, c.want) {
					t.Errorf("CreateItem = %v, want %v", err, c.want)
				}
			})
		}
	})

	t.Run("reads back what was written", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		made := create(t, s, tracker.NewItem{Title: "It breaks", Body: "steps"})
		got, err := s.Item(ctx, made.ID)
		if err != nil {
			t.Fatalf("Item: %v", err)
		}
		if got.Title != "It breaks" || got.Body != "steps" {
			t.Errorf("Item = %+v, want the values as written", got)
		}
	})

	t.Run("reports a missing item", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		if _, err := s.Item(ctx, "nope"); !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("Item = %v, want ErrNotFound", err)
		}
	})

	t.Run("refuses to delete an item with children", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		parent := create(t, s, tracker.NewItem{})
		child(t, s, parent.ID)
		if err := s.DeleteItem(ctx, parent.ID, parent.Version, human("vk")); !errors.Is(err, tracker.ErrHasChildren) {
			t.Errorf("DeleteItem = %v, want ErrHasChildren", err)
		}
	})

	t.Run("deletes a leaf", func(t *testing.T) {
		s, ctx := fixture(t, newStore)
		item := create(t, s, tracker.NewItem{})
		if err := s.DeleteItem(ctx, item.ID, item.Version, human("vk")); err != nil {
			t.Fatalf("DeleteItem: %v", err)
		}
		if _, err := s.Item(ctx, item.ID); !errors.Is(err, tracker.ErrNotFound) {
			t.Errorf("Item after delete = %v, want ErrNotFound", err)
		}
	})
}
