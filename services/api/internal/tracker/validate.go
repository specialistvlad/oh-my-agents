package tracker

// Validator is implemented by every level of the schema. Validate checks the
// level itself and then everything beneath it, so validating the root
// validates the whole tree: a [Schema] validates its types, a type validates
// its fields, statuses and transitions, each of those validates itself.
//
// Nothing here calls down into storage. Validation is a pure function of the
// values in hand, which is what lets it run before a write, inside a test, or
// against a config file that has never been saved.
//
// This is a reusable helper, not a mandatory layer. A store built on
// something with no constraint machinery of its own — the filesystem — is
// expected to lean on it. A store built on something that enforces natively
// may ignore it entirely. See [Store] for who owes what.
type Validator interface {
	Validate() error
}

// Schema is the whole configuration: every [ItemType] the tracker knows.
// It is the root of the validation tree and the authority content rules are
// checked against.
//
// Schema will carry, alongside [Validator]:
//
//	Type(TypeKey) (ItemType, bool)         look up one type
//	ValidateItem(Item) error               item against its type's fields
//	ValidateNew(NewItem) error             creation, before an ID exists
//	ValidatePatch(Item, Patch) error       an edit against the current item
//	ValidateTransition(ItemType, from, to StatusKey, Item) error
//
// None of these are implemented yet.
type Schema struct {
	Types []ItemType
}
