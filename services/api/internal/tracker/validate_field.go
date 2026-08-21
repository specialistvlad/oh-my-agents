package tracker

import "fmt"

// Validate checks the field definition is internally consistent: a known
// kind, options exactly where options belong, and a default that fits.
func (f FieldDef) Validate() error {
	if f.Key == "" || f.Name == "" {
		return fmt.Errorf("%w: field %q needs a key and a name", ErrInvalidSchema, f.Key)
	}
	if reserved(f.Key) {
		return fmt.Errorf("%w: field %q uses the reserved @ prefix", ErrReservedKey, f.Key)
	}
	if !f.Kind.valid() {
		return fmt.Errorf("%w: field %q has unknown kind %q", ErrInvalidSchema, f.Key, f.Kind)
	}
	if err := f.validateOptions(); err != nil {
		return err
	}
	if f.Kind != KindItem && len(f.ItemTypes) > 0 {
		return fmt.Errorf("%w: field %q restricts item types but is not an item reference", ErrInvalidSchema, f.Key)
	}
	return f.validateDefault()
}

// validateOptions enforces that select kinds carry choices and nothing else
// does, and that the choices are distinct.
func (f FieldDef) validateOptions() error {
	selects := f.Kind == KindSelect || f.Kind == KindMultiSelect
	switch {
	case selects && len(f.Options) == 0:
		return fmt.Errorf("%w: field %q is a select with no options", ErrInvalidSchema, f.Key)
	case !selects && len(f.Options) > 0:
		return fmt.Errorf("%w: field %q declares options but is a %s", ErrInvalidSchema, f.Key, f.Kind)
	}
	seen := make(map[OptionKey]struct{}, len(f.Options))
	for _, o := range f.Options {
		if o.Key == "" || o.Name == "" {
			return fmt.Errorf("%w: field %q has an option missing a key or name", ErrInvalidSchema, f.Key)
		}
		if _, dup := seen[o.Key]; dup {
			return fmt.Errorf("%w: field %q declares option %q twice", ErrInvalidSchema, f.Key, o.Key)
		}
		seen[o.Key] = struct{}{}
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
		return fmt.Errorf("%w: field %q is a %s, got a %s", ErrKindMismatch, f.Key, f.Kind, v.Kind())
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

func (f FieldDef) acceptsOption(o OptionKey) error {
	for _, declared := range f.Options {
		if declared.Key == o {
			return nil
		}
	}
	return fmt.Errorf("%w: field %q has no option %q", ErrUnknownOption, f.Key, o)
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
