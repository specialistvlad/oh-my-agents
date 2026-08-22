package tracker

// FieldDef declares one custom field on one [ItemType].
type FieldDef struct {
	ID          FieldID   `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Kind        FieldKind `json:"kind"`
	// Required rejects an item that has no value for this field. A
	// transition's RequiredFields can demand a value later in the
	// lifecycle instead, for fields that cannot be known at creation.
	Required bool `json:"required"`
	// Options are the permitted choices for [KindSelect] and
	// [KindMultiSelect], and must be empty for every other kind.
	Options []Option `json:"options"`
	// Default is applied at creation when the caller supplies no value.
	Default *Value `json:"default"`
	// ItemTypes narrows [KindItem] to particular types, e.g. a "blocked by"
	// field that may only point at bugs. Empty means any type.
	ItemTypes []TypeID `json:"item_types"`
}

// FieldKind is the type of a field's value. It determines what [Value.Raw]
// holds; see [Value] for the mapping.
type FieldKind string

// The field kinds.
const (
	// KindText is a single-line string.
	KindText FieldKind = "text"
	// KindMarkdown is a multi-line string rendered as Markdown.
	KindMarkdown FieldKind = "markdown"
	// KindNumber is a float64, covering integers and decimals alike.
	KindNumber FieldKind = "number"
	// KindBool is a checkbox.
	KindBool FieldKind = "bool"
	// KindDate is an instant in UTC.
	KindDate FieldKind = "date"
	// KindDuration is a span, for estimates and time spent.
	KindDuration FieldKind = "duration"
	// KindSelect is one choice from Options.
	KindSelect FieldKind = "select"
	// KindMultiSelect is zero or more choices from Options.
	KindMultiSelect FieldKind = "multi_select"
	// KindActor is a reference to a human, an agent or the system —
	// assignee, reviewer, owner.
	KindActor FieldKind = "actor"
	// KindItem is a reference to another item.
	KindItem FieldKind = "item"
	// KindURL is an absolute URL.
	KindURL FieldKind = "url"
)

// Option is one choice of a select field. Key is stable; Name is the label
// and may be edited freely.
type Option struct {
	ID   OptionID `json:"id"`
	Name string   `json:"name"`
}

// Reserved field keys name the built-in parts of an [Item] so that a
// [Change] can describe a title or status edit with the same shape it uses
// for custom fields. The "@" prefix is reserved: a [FieldDef] may not use it.
const (
	// FieldTitle is the item's title.
	FieldTitle FieldID = "@title"
	// FieldBody is the item's description.
	FieldBody FieldID = "@body"
	// FieldStatus is the item's status.
	FieldStatus FieldID = "@status"
	// FieldParent is the item's parent.
	FieldParent FieldID = "@parent"
)
