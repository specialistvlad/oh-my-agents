package tracker

// ItemType defines one class of work — epic, feature, user story,
// requirement, release, bug. It is data, written and edited at runtime, so a
// new class of work costs a write rather than a deploy.
//
// A type owns its own fields and its own statuses. Two types share nothing
// but the shape of this struct: a bug's statuses and a release's statuses are
// unrelated sets, and neither can be used on the other's items.
type ItemType struct {
	ID          TypeID `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// Fields are the custom fields items of this type carry. An item may
	// hold values only for keys declared here.
	Fields []FieldDef `json:"fields"`
	// Statuses are the columns items of this type move through. The set is
	// per-type and arbitrary; meaning is carried by each status's category,
	// never by its key or name.
	Statuses []Status `json:"statuses"`
	// Initial is the status a newly created item enters. It must name one
	// of Statuses. Creation does not go through Transitions.
	Initial StatusID `json:"initial"`
	// Transitions is the allowed-moves graph, and it is always enforced: a
	// move not listed here is rejected. A type with no transitions can be
	// created but never moved, which is a configuration error rather than a
	// special case.
	Transitions []Transition `json:"transitions"`
}

// Status is one column of one type's workflow.
//
// Key and Name are independent so a column can be relabelled without
// invalidating stored items or breaking logic that keys off it.
type Status struct {
	ID       StatusID       `json:"id"`
	Name     string         `json:"name"`
	Category StatusCategory `json:"category"`
}

// StatusCategory is the fixed semantic axis behind user-defined statuses.
//
// Statuses themselves are arbitrary — a team may invent "Awaiting Design
// Review" — but every one of them maps onto a category the system
// understands. Logic that needs to know whether work is finished asks the
// category, never the key, so renaming a column cannot break orchestration.
type StatusCategory string

// The status categories. [CategoryDone] and [CategoryCanceled] are the
// resolved ones: an item in either is settled, and only then may its parent
// resolve.
const (
	// CategoryBacklog is accepted but not started.
	CategoryBacklog StatusCategory = "backlog"
	// CategoryActive is in progress.
	CategoryActive StatusCategory = "active"
	// CategoryBlocked is started but waiting on something external.
	CategoryBlocked StatusCategory = "blocked"
	// CategoryDone is finished successfully.
	CategoryDone StatusCategory = "done"
	// CategoryCanceled is finished without being completed. It counts as
	// resolved: a canceled child does not hold its parent open.
	CategoryCanceled StatusCategory = "canceled"
)

// Transition is one permitted move in a type's workflow.
type Transition struct {
	From StatusID `json:"from"`
	To   StatusID `json:"to"`
	// RequiredFields must hold a value before the move is allowed. This is
	// how a workflow demands, say, a resolution note on the way to done.
	RequiredFields []FieldID `json:"required_fields"`
}
