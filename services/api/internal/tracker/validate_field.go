package tracker

import "fmt"

// Validate checks the field definition is internally consistent: a known
// kind, options exactly where options belong, and a default that fits.
func (f FieldDef) Validate() error {
	if reserved(f.ID) {
		return fmt.Errorf("%w: field %q uses the reserved @ prefix", ErrReservedKey, f.ID)
	}
	if err := validID("field", string(f.ID)); err != nil {
		return err
	}
	if f.Name == "" {
		return fmt.Errorf("%w: field %q needs a name", ErrInvalidSchema, f.ID)
	}
	if !f.Kind.valid() {
		return fmt.Errorf("%w: field %q has unknown kind %q", ErrInvalidSchema, f.ID, f.Kind)
	}
	if err := f.validateOptions(); err != nil {
		return err
	}
	if f.Kind != KindItem && len(f.ItemTypes) > 0 {
		return fmt.Errorf("%w: field %q restricts item types but is not an item reference", ErrInvalidSchema, f.ID)
	}
	return f.validateDefault()
}

// validateOptions enforces that select kinds carry choices and nothing else
// does, and that the choices are distinct.
func (f FieldDef) validateOptions() error {
	selects := f.Kind == KindSelect || f.Kind == KindMultiSelect
	switch {
	case selects && len(f.Options) == 0:
		return fmt.Errorf("%w: field %q is a select with no options", ErrInvalidSchema, f.ID)
	case !selects && len(f.Options) > 0:
		return fmt.Errorf("%w: field %q declares options but is a %s", ErrInvalidSchema, f.ID, f.Kind)
	}
	seen := make(map[OptionID]struct{}, len(f.Options))
	for _, o := range f.Options {
		if err := o.Validate(); err != nil {
			return fmt.Errorf("field %q: %w", f.ID, err)
		}
		if _, dup := seen[o.ID]; dup {
			return fmt.Errorf("%w: field %q declares option %q twice", ErrInvalidSchema, f.ID, o.ID)
		}
		seen[o.ID] = struct{}{}
	}
	return nil
}

func (f FieldDef) validateDefault() error {
	if f.Default == nil {
		return nil
	}
	return f.Accepts(*f.Default)
}

// Accepts reports whether a value may be stored in this field, checking both
// that the kinds agree and, for select kinds, that every chosen option is one
// this field declares.
//
// The zero [Value] is accepted here and means "no value"; whether the field
// may be left empty is [FieldDef.Required]'s question, asked elsewhere.
func (f FieldDef) Accepts(v Value) error {
	if v.IsZero() {
		return nil
	}
	if v.Kind() != f.Kind {
		return fmt.Errorf("%w: field %q is a %s, got a %s", ErrKindMismatch, f.ID, f.Kind, v.Kind())
	}
	switch f.Kind {
	case KindSelect:
		o, _ := v.Select()
		return f.acceptsOption(o)
	case KindMultiSelect:
		chosen, _ := v.MultiSelect()
		for _, o := range chosen {
			if err := f.acceptsOption(o); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func (f FieldDef) acceptsOption(o OptionID) error {
	for _, declared := range f.Options {
		if declared.ID == o {
			return nil
		}
	}
	return fmt.Errorf("%w: field %q has no option %q", ErrUnknownOption, f.ID, o)
}

func (k FieldKind) valid() bool {
	switch k {
	case KindText, KindMarkdown, KindNumber, KindBool, KindDate, KindDuration,
		KindSelect, KindMultiSelect, KindActor, KindItem, KindURL:
		return true
	default:
		return false
	}
}

// Validate checks the option is nameable and addressable.
func (o Option) Validate() error {
	if err := validID("option", string(o.ID)); err != nil {
		return err
	}
	if o.Name == "" {
		return fmt.Errorf("%w: option %q needs a name", ErrInvalidSchema, o.ID)
	}
	return nil
}
