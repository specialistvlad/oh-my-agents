package tracker

import (
	"encoding/json"
	"fmt"
	"time"
)

// Value encodes as its kind and its payload:
//
//	{"kind":"select","value":"high-5n8p"}
//	{"kind":"date","value":"2026-08-21T12:00:00Z"}
//
// JSON in the domain is a deliberate exception to ADR-0002's rule that
// encoding stops at the adapter. Every adapter and every edge needs this
// exact translation, and a [Value] cannot be rebuilt from outside the package
// — its constructors are typed and its fields unexported, which is what makes
// kind and payload agree. Writing it once here is the alternative to writing
// it identically in each store and each wire format, and getting it subtly
// different in one of them.
type wire struct {
	Kind  FieldKind       `json:"kind"`
	Value json.RawMessage `json:"value,omitempty"`
}

// MarshalJSON implements [json.Marshaler]. The zero Value encodes as null,
// which is how an absent field survives a round trip.
func (v Value) MarshalJSON() ([]byte, error) {
	if v.IsZero() {
		return []byte("null"), nil
	}
	payload, err := json.Marshal(v.encodable())
	if err != nil {
		return nil, fmt.Errorf("tracker: encode %s value: %w", v.kind, err)
	}
	return json.Marshal(wire{Kind: v.kind, Value: payload})
}

// encodable renders the payload in the most readable form the kind allows,
// because the point of a filesystem store is that a person can read it.
func (v Value) encodable() any {
	switch v.kind {
	case KindDate:
		t, _ := v.Date()
		return t.Format(time.RFC3339Nano)
	case KindDuration:
		d, _ := v.Duration()
		return d.String()
	default:
		return v.raw
	}
}

// UnmarshalJSON implements [json.Unmarshaler].
func (v *Value) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*v = Value{}
		return nil
	}
	var w wire
	if err := json.Unmarshal(data, &w); err != nil {
		return fmt.Errorf("tracker: decode value: %w", err)
	}
	rebuilt, err := rebuild(w.Kind, w.Value)
	if err != nil {
		return err
	}
	*v = rebuilt
	return nil
}

// rebuild reconstructs a value from its kind and encoded payload, refusing
// anything the kind cannot hold. A stored file edited by hand into an
// impossible shape fails here rather than becoming a Value whose kind lies.
func rebuild(kind FieldKind, payload json.RawMessage) (Value, error) {
	into := func(target any) error {
		if err := json.Unmarshal(payload, target); err != nil {
			return fmt.Errorf("%w: %s payload: %w", ErrKindMismatch, kind, err)
		}
		return nil
	}
	switch kind {
	case KindText, KindMarkdown, KindURL:
		var s string
		err := into(&s)
		return Value{kind: kind, raw: s}, err
	case KindNumber:
		var f float64
		err := into(&f)
		return Number(f), err
	case KindBool:
		var b bool
		err := into(&b)
		return Bool(b), err
	case KindDate:
		var s string
		if err := into(&s); err != nil {
			return Value{}, err
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return Value{}, fmt.Errorf("%w: date %q: %w", ErrKindMismatch, s, err)
		}
		return Date(t), nil
	case KindDuration:
		var s string
		if err := into(&s); err != nil {
			return Value{}, err
		}
		d, err := time.ParseDuration(s)
		if err != nil {
			return Value{}, fmt.Errorf("%w: duration %q: %w", ErrKindMismatch, s, err)
		}
		return Duration(d), nil
	case KindSelect:
		var o OptionID
		err := into(&o)
		return Select(o), err
	case KindMultiSelect:
		var options []OptionID
		err := into(&options)
		return MultiSelect(options...), err
	case KindActor:
		var a ActorRef
		err := into(&a)
		return Actor(a), err
	case KindItem:
		var id ItemID
		err := into(&id)
		return ItemRef(id), err
	default:
		return Value{}, fmt.Errorf("%w: unknown kind %q", ErrKindMismatch, kind)
	}
}
