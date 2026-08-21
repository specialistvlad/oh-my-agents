package tracker

// Value is one field's value. Kind says how to read Raw:
//
//	KindText, KindMarkdown, KindURL   string
//	KindNumber                        float64
//	KindBool                          bool
//	KindDate                          time.Time
//	KindDuration                      time.Duration
//	KindSelect                        OptionKey
//	KindMultiSelect                   []OptionKey
//	KindActor                         ActorRef
//	KindItem                          ItemID
//
// Kind and Raw must agree. Nothing in this package enforces that yet:
// typed constructors and accessors — tracker.Text("…"), v.Text() (string,
// bool) — arrive with the schema validator, and until then the invariant
// holds by convention. Adapters translating at their boundary read Raw
// directly, which is what it is exported for.
type Value struct {
	Kind FieldKind
	Raw  any
}
