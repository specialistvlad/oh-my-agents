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

// Every level of the schema implements it. The assertions are here so that a
// level losing its Validate method breaks the build rather than quietly
// dropping out of the recursion.
var (
	_ Validator = Schema{}
	_ Validator = ItemType{}
	_ Validator = FieldDef{}
	_ Validator = Status{}
	_ Validator = Transition{}
	_ Validator = Option{}
)

// Schema is the whole configuration: every [ItemType] the tracker knows.
// It is the root of the validation tree and the authority content rules are
// checked against.
//
// Alongside [Validator], Schema answers the content questions:
// [Schema.Type], [Schema.ValidateItem], [Schema.ValidateNew],
// [Schema.ValidatePatch] and [Schema.ValidateTransition].
//
// All of them are pure. The rules that need to see other items — the tree
// staying acyclic, a parent not resolving before its descendants — cannot be
// answered from a schema alone and belong to the store, which is where
// [Store] documents them.
type Schema struct {
	Types []ItemType `json:"types"`
}
