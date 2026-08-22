package trackertest

import "github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"

// The identifiers the fixture type is built from.
//
// They are minted-shaped — a readable stem and a suffix — rather than the
// words a person would type, because that is what the system stores
// (ADR-0009). Naming them here keeps the suite readable without any test
// depending on an id's spelling, which is the property the ids exist to
// protect: rename "Bug" to "Defect" and only the Name below changes.
const (
	TypeBug tracker.TypeID = "bug-9c2x"

	FieldSummary    tracker.FieldID = "summary-3k1p"
	FieldSeverity   tracker.FieldID = "severity-8m2q"
	FieldResolution tracker.FieldID = "resolution-5v7d"

	StatusOpen    tracker.StatusID = "open-1a4f"
	StatusDoing   tracker.StatusID = "doing-6b2c"
	StatusFixed   tracker.StatusID = "fixed-7d9e"
	StatusDropped tracker.StatusID = "dropped-2f8h"

	OptionLow    tracker.OptionID = "low-4j6k"
	OptionMedium tracker.OptionID = "medium-9l3m"
	OptionHigh   tracker.OptionID = "high-5n8p"
)

// BugType is the type the suite configures: a required field, a select with a
// default, a field only a late transition demands, and a workflow that is a
// graph rather than a line.
//
// It is exported so an adapter's own tests can build on the same shape the
// suite assumes.
func BugType() tracker.ItemType {
	severity := tracker.Select(OptionMedium)
	return tracker.ItemType{
		ID:   TypeBug,
		Name: "Bug",
		Fields: []tracker.FieldDef{
			{ID: FieldSummary, Name: "Summary", Kind: tracker.KindText, Required: true},
			{ID: FieldSeverity, Name: "Severity", Kind: tracker.KindSelect, Default: &severity, Options: []tracker.Option{
				{ID: OptionLow, Name: "Low"},
				{ID: OptionMedium, Name: "Medium"},
				{ID: OptionHigh, Name: "High"},
			}},
			{ID: FieldResolution, Name: "Resolution", Kind: tracker.KindMarkdown},
		},
		Statuses: []tracker.Status{
			{ID: StatusOpen, Name: "Open", Category: tracker.CategoryBacklog},
			{ID: StatusDoing, Name: "Doing", Category: tracker.CategoryActive},
			{ID: StatusFixed, Name: "Fixed", Category: tracker.CategoryDone},
			{ID: StatusDropped, Name: "Dropped", Category: tracker.CategoryCanceled},
		},
		Initial: StatusOpen,
		Transitions: []tracker.Transition{
			{From: StatusOpen, To: StatusDoing},
			{From: StatusDoing, To: StatusOpen},
			{From: StatusDoing, To: StatusFixed, RequiredFields: []tracker.FieldID{FieldResolution}},
			{From: StatusFixed, To: StatusOpen},
			{From: StatusOpen, To: StatusDropped},
			{From: StatusDropped, To: StatusOpen},
		},
	}
}
