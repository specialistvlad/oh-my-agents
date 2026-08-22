package tracker

import (
	"slices"
	"time"
)

// The accessors each report whether the value is of the kind asked for. A
// false means the caller asked the wrong question, not that the value is
// empty — check [Value.IsZero] for that.

// String returns the text of a [KindText], [KindMarkdown] or [KindURL] value.
// The three share a representation, and a caller that only wants to render
// text should not have to care which of them it has.
func (v Value) String() (string, bool) {
	switch v.kind {
	case KindText, KindMarkdown, KindURL:
		s, ok := v.raw.(string)
		return s, ok
	default:
		return "", false
	}
}

// Number returns the value of a [KindNumber].
func (v Value) Number() (float64, bool) {
	f, ok := v.raw.(float64)
	return f, ok && v.kind == KindNumber
}

// Bool returns the value of a [KindBool].
func (v Value) Bool() (bool, bool) {
	b, ok := v.raw.(bool)
	return b, ok && v.kind == KindBool
}

// Date returns the value of a [KindDate], in UTC.
func (v Value) Date() (time.Time, bool) {
	t, ok := v.raw.(time.Time)
	return t, ok && v.kind == KindDate
}

// Duration returns the value of a [KindDuration].
func (v Value) Duration() (time.Duration, bool) {
	d, ok := v.raw.(time.Duration)
	return d, ok && v.kind == KindDuration
}

// Select returns the chosen option of a [KindSelect].
func (v Value) Select() (OptionID, bool) {
	o, ok := v.raw.(OptionID)
	return o, ok && v.kind == KindSelect
}

// MultiSelect returns a copy of the chosen options of a [KindMultiSelect].
func (v Value) MultiSelect() ([]OptionID, bool) {
	o, ok := v.raw.([]OptionID)
	if !ok || v.kind != KindMultiSelect {
		return nil, false
	}
	return slices.Clone(o), true
}

// Actor returns the actor of a [KindActor].
func (v Value) Actor() (ActorRef, bool) {
	a, ok := v.raw.(ActorRef)
	return a, ok && v.kind == KindActor
}

// Item returns the item referenced by a [KindItem].
func (v Value) Item() (ItemID, bool) {
	id, ok := v.raw.(ItemID)
	return id, ok && v.kind == KindItem
}

// Equal reports whether two values hold the same kind and the same payload.
// It is what [FieldMatch] is resolved with, so every backend agrees on what
// equality means rather than inheriting its storage engine's opinion.
func (v Value) Equal(other Value) bool {
	if v.kind != other.kind {
		return false
	}
	switch v.kind {
	case "":
		return true
	case KindMultiSelect:
		a, _ := v.raw.([]OptionID)
		b, _ := other.raw.([]OptionID)
		return slices.Equal(a, b)
	case KindDate:
		a, _ := v.raw.(time.Time)
		b, _ := other.raw.(time.Time)
		return a.Equal(b)
	case KindText, KindMarkdown, KindURL, KindNumber, KindBool,
		KindDuration, KindSelect, KindActor, KindItem:
		return v.raw == other.raw
	default:
		return v.raw == other.raw
	}
}
