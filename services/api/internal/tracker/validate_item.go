package tracker

import "fmt"

// ValidateItem checks a stored item against its type: the type exists, the
// status is one it declares, every field key is declared, every value fits
// its field, and nothing required is missing.
func (s Schema) ValidateItem(item Item) error {
	t, ok := s.Type(item.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownType, item.Type)
	}
	if _, ok := t.Status(item.Status); !ok {
		return fmt.Errorf("%w: type %q has no status %q", ErrUnknownStatus, t.ID, item.Status)
	}
	if err := s.validateFields(t, item.Fields); err != nil {
		return err
	}
	return requireAll(t, item.Fields)
}

// ValidateNew checks a creation. Required fields are judged after defaults
// are applied, because a field with a default is never actually missing.
func (s Schema) ValidateNew(n NewItem) error {
	t, ok := s.Type(n.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownType, n.Type)
	}
	if err := s.validateFields(t, n.Fields); err != nil {
		return err
	}
	return requireAll(t, t.ApplyDefaults(n.Fields))
}

// ValidatePatch checks an edit against the item it applies to. It covers the
// fields being written and, when the status moves, the workflow — but not the
// tree, whose rules need to look at other items and so belong to the store.
func (s Schema) ValidatePatch(current Item, p Patch) error {
	t, ok := s.Type(current.Type)
	if !ok {
		return fmt.Errorf("%w: %q", ErrUnknownType, current.Type)
	}
	if p.Parent != nil && p.ClearParent {
		return fmt.Errorf("%w: patch both sets and clears the parent", ErrInvalidSchema)
	}
	after, err := s.apply(t, current, p)
	if err != nil {
		return err
	}
	if err := requireAll(t, after.Fields); err != nil {
		return err
	}
	if p.Status == nil || *p.Status == current.Status {
		return nil
	}
	return s.ValidateTransition(t, current.Status, *p.Status, after)
}

// ValidateTransition checks one status move against the type's workflow. It
// is always enforced: a move the type does not declare is refused, and a
// transition's required fields must hold values in the item as it will be
// after the edit, not as it was before.
func (s Schema) ValidateTransition(t ItemType, from, to StatusID, after Item) error {
	if _, ok := t.Status(to); !ok {
		return fmt.Errorf("%w: type %q has no status %q", ErrUnknownStatus, t.ID, to)
	}
	tr, ok := t.Transition(from, to)
	if !ok {
		return fmt.Errorf("%w: type %q does not allow %s -> %s", ErrTransitionNotAllowed, t.ID, from, to)
	}
	for _, key := range tr.RequiredFields {
		if v, held := after.Fields[key]; !held || v.IsZero() {
			return fmt.Errorf("%w: %s -> %s needs field %q", ErrFieldRequired, from, to, key)
		}
	}
	return nil
}

// validateFields checks that every key is declared by the type and every
// value fits the field it is being written to.
func (s Schema) validateFields(t ItemType, fields map[FieldID]Value) error {
	for key, v := range fields {
		f, ok := t.Field(key)
		if !ok {
			return fmt.Errorf("%w: type %q has no field %q", ErrUnknownField, t.ID, key)
		}
		if err := f.Accepts(v); err != nil {
			return err
		}
	}
	return nil
}

// apply produces the item a patch would result in, so that required fields
// and transitions are judged against the outcome rather than the starting
// point. It does not touch the store and does not check the tree.
func (s Schema) apply(t ItemType, current Item, p Patch) (Item, error) {
	after := current
	after.Fields = make(map[FieldID]Value, len(current.Fields))
	for k, v := range current.Fields {
		after.Fields[k] = v
	}
	for key, v := range p.Fields {
		f, ok := t.Field(key)
		if !ok {
			return Item{}, fmt.Errorf("%w: type %q has no field %q", ErrUnknownField, t.ID, key)
		}
		if v == nil {
			delete(after.Fields, key)
			continue
		}
		if err := f.Accepts(*v); err != nil {
			return Item{}, err
		}
		after.Fields[key] = *v
	}
	if p.Status != nil {
		after.Status = *p.Status
	}
	return after, nil
}

// ApplyDefaults returns the fields a new item would carry: whatever the
// caller supplied, plus a default for every declared field it left out.
func (t ItemType) ApplyDefaults(fields map[FieldID]Value) map[FieldID]Value {
	out := make(map[FieldID]Value, len(fields)+len(t.Fields))
	for k, v := range fields {
		out[k] = v
	}
	for _, f := range t.Fields {
		if _, held := out[f.ID]; held || f.Default == nil {
			continue
		}
		out[f.ID] = *f.Default
	}
	return out
}

// requireAll checks every required field holds a value.
func requireAll(t ItemType, fields map[FieldID]Value) error {
	for _, f := range t.Fields {
		if !f.Required {
			continue
		}
		if v, held := fields[f.ID]; !held || v.IsZero() {
			return fmt.Errorf("%w: type %q needs field %q", ErrFieldRequired, t.ID, f.ID)
		}
	}
	return nil
}
