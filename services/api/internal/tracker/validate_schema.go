package tracker

import (
	"fmt"
	"strings"
)

// Validate checks every type in the schema, and that no two share a key.
func (s Schema) Validate() error {
	seen := make(map[TypeKey]struct{}, len(s.Types))
	for _, t := range s.Types {
		if _, dup := seen[t.Key]; dup {
			return fmt.Errorf("%w: duplicate type %q", ErrInvalidSchema, t.Key)
		}
		seen[t.Key] = struct{}{}
		if err := t.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Type looks up one type by key.
func (s Schema) Type(key TypeKey) (ItemType, bool) {
	for _, t := range s.Types {
		if t.Key == key {
			return t, true
		}
	}
	return ItemType{}, false
}

// Validate checks the type and everything beneath it: its fields, its
// statuses, and that its workflow refers only to things it declares.
func (t ItemType) Validate() error {
	if t.Key == "" || t.Name == "" {
		return fmt.Errorf("%w: type %q needs a key and a name", ErrInvalidSchema, t.Key)
	}
	if err := t.validateFields(); err != nil {
		return err
	}
	return t.validateWorkflow()
}

func (t ItemType) validateFields() error {
	seen := make(map[FieldKey]struct{}, len(t.Fields))
	for _, f := range t.Fields {
		if _, dup := seen[f.Key]; dup {
			return fmt.Errorf("%w: type %q declares field %q twice", ErrInvalidSchema, t.Key, f.Key)
		}
		seen[f.Key] = struct{}{}
		if err := f.Validate(); err != nil {
			return fmt.Errorf("type %q: %w", t.Key, err)
		}
	}
	return nil
}

// validateWorkflow checks the statuses, the initial status and the transition
// graph together, because each is only meaningful against the others.
func (t ItemType) validateWorkflow() error {
	if len(t.Statuses) == 0 {
		return fmt.Errorf("%w: type %q declares no statuses", ErrInvalidSchema, t.Key)
	}
	statuses := make(map[StatusKey]struct{}, len(t.Statuses))
	for _, st := range t.Statuses {
		if _, dup := statuses[st.Key]; dup {
			return fmt.Errorf("%w: type %q declares status %q twice", ErrInvalidSchema, t.Key, st.Key)
		}
		statuses[st.Key] = struct{}{}
		if err := st.Validate(); err != nil {
			return fmt.Errorf("type %q: %w", t.Key, err)
		}
	}
	if _, ok := statuses[t.Initial]; !ok {
		return fmt.Errorf("%w: type %q starts at %q, which it does not declare", ErrInvalidSchema, t.Key, t.Initial)
	}
	return t.validateTransitions(statuses)
}

func (t ItemType) validateTransitions(statuses map[StatusKey]struct{}) error {
	fields := make(map[FieldKey]struct{}, len(t.Fields))
	for _, f := range t.Fields {
		fields[f.Key] = struct{}{}
	}
	// Transition carries a slice, so it cannot be a map key; the pair of
	// endpoints is what has to be unique anyway.
	type edge struct{ from, to StatusKey }
	seen := make(map[edge]struct{}, len(t.Transitions))
	for _, tr := range t.Transitions {
		e := edge{from: tr.From, to: tr.To}
		if _, dup := seen[e]; dup {
			return fmt.Errorf("%w: type %q declares %s->%s twice", ErrInvalidSchema, t.Key, tr.From, tr.To)
		}
		seen[e] = struct{}{}
		for _, end := range []StatusKey{tr.From, tr.To} {
			if _, ok := statuses[end]; !ok {
				return fmt.Errorf("%w: type %q transitions via %q, which it does not declare", ErrInvalidSchema, t.Key, end)
			}
		}
		for _, req := range tr.RequiredFields {
			if _, ok := fields[req]; !ok {
				return fmt.Errorf("%w: type %q requires field %q on %s->%s, which it does not declare",
					ErrInvalidSchema, t.Key, req, tr.From, tr.To)
			}
		}
	}
	return nil
}

// CanTransition reports whether the type's workflow permits this move.
// Creation does not go through here: a new item enters [ItemType.Initial]
// directly.
func (t ItemType) CanTransition(from, to StatusKey) bool {
	for _, tr := range t.Transitions {
		if tr.From == from && tr.To == to {
			return true
		}
	}
	return false
}

// Transition returns the declared move, if there is one.
func (t ItemType) Transition(from, to StatusKey) (Transition, bool) {
	for _, tr := range t.Transitions {
		if tr.From == from && tr.To == to {
			return tr, true
		}
	}
	return Transition{}, false
}

// Status looks up one status by key.
func (t ItemType) Status(key StatusKey) (Status, bool) {
	for _, st := range t.Statuses {
		if st.Key == key {
			return st, true
		}
	}
	return Status{}, false
}

// Field looks up one field by key.
func (t ItemType) Field(key FieldKey) (FieldDef, bool) {
	for _, f := range t.Fields {
		if f.Key == key {
			return f, true
		}
	}
	return FieldDef{}, false
}

// Validate checks the status names a known category.
func (s Status) Validate() error {
	if s.Key == "" || s.Name == "" {
		return fmt.Errorf("%w: status %q needs a key and a name", ErrInvalidSchema, s.Key)
	}
	if !s.Category.valid() {
		return fmt.Errorf("%w: status %q has unknown category %q", ErrInvalidSchema, s.Key, s.Category)
	}
	return nil
}

// Validate checks both ends of the move are named.
func (t Transition) Validate() error {
	if t.From == "" || t.To == "" {
		return fmt.Errorf("%w: transition needs both ends", ErrInvalidSchema)
	}
	return nil
}

// Resolved reports whether work in this category is settled. It is the single
// place the resolution gate's question is answered, so "is this finished" has
// one definition rather than one per caller.
func (c StatusCategory) Resolved() bool {
	return c == CategoryDone || c == CategoryCanceled
}

func (c StatusCategory) valid() bool {
	switch c {
	case CategoryBacklog, CategoryActive, CategoryBlocked, CategoryDone, CategoryCanceled:
		return true
	default:
		return false
	}
}

// reserved reports whether a field key uses the "@" namespace that [Change]
// relies on for built-in parts of an item.
func reserved(k FieldKey) bool { return strings.HasPrefix(string(k), "@") }
