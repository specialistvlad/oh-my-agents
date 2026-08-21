package tracker

import "time"

// Value is one field's value: a kind and a payload that always agree.
//
// The fields are unexported and every constructor is typed, so a Value that
// claims to be a number and holds a string cannot be built. That invariant is
// the whole point of the type — it is what lets the rest of the system treat
// a runtime-defined schema as if it were statically typed.
//
// The zero Value means "no value", which is distinct from a value that
// happens to be empty: an unset text field and a field holding "" are not the
// same thing, and [Value.IsZero] tells them apart.
type Value struct {
	kind FieldKind
	raw  any
}

// Text returns a [KindText] value.
func Text(s string) Value { return Value{kind: KindText, raw: s} }

// Markdown returns a [KindMarkdown] value.
func Markdown(s string) Value { return Value{kind: KindMarkdown, raw: s} }

// URL returns a [KindURL] value.
func URL(s string) Value { return Value{kind: KindURL, raw: s} }

// Number returns a [KindNumber] value.
func Number(f float64) Value { return Value{kind: KindNumber, raw: f} }

// Bool returns a [KindBool] value.
func Bool(b bool) Value { return Value{kind: KindBool, raw: b} }

// Date returns a [KindDate] value, normalized to UTC so that two instants
// describing the same moment compare equal.
func Date(t time.Time) Value { return Value{kind: KindDate, raw: t.UTC()} }

// Duration returns a [KindDuration] value.
func Duration(d time.Duration) Value { return Value{kind: KindDuration, raw: d} }

// Select returns a [KindSelect] value.
func Select(o OptionKey) Value { return Value{kind: KindSelect, raw: o} }

// MultiSelect returns a [KindMultiSelect] value. The options are copied, so a
// caller reusing its slice cannot reach inside the value afterwards.
func MultiSelect(options ...OptionKey) Value {
	out := make([]OptionKey, len(options))
	copy(out, options)
	return Value{kind: KindMultiSelect, raw: out}
}

// Actor returns a [KindActor] value.
func Actor(a ActorRef) Value { return Value{kind: KindActor, raw: a} }

// ItemRef returns a [KindItem] value pointing at another item.
func ItemRef(id ItemID) Value { return Value{kind: KindItem, raw: id} }

// Kind reports what this value holds. The zero Value reports the empty kind.
func (v Value) Kind() FieldKind { return v.kind }

// IsZero reports whether this is the zero Value — no value at all, as opposed
// to a value that is empty.
func (v Value) IsZero() bool { return v.kind == "" }

// Raw exposes the payload for adapters translating at their own boundary,
// which is the only place that should need it. Everything else uses the typed
// accessors, which cannot silently return the wrong shape.
//
// The mapping is fixed: string for text, markdown and URL; float64 for
// number; bool; [time.Time] for date; [time.Duration]; [OptionKey] for
// select; []OptionKey for multi-select; [ActorRef]; [ItemID] for an item
// reference.
func (v Value) Raw() any { return v.raw }
