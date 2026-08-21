package trackertest

import "github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"

// BugType is the type the suite configures: a required field, a select with
// a default, a field only a late transition demands, and a workflow that is a
// graph rather than a line.
//
// It is exported so an adapter's own tests can build on the same shape the
// suite assumes.
func BugType() tracker.ItemType {
	severity := tracker.Select("medium")
	return tracker.ItemType{
		Key:  "bug",
		Name: "Bug",
		Fields: []tracker.FieldDef{
			{Key: "summary", Name: "Summary", Kind: tracker.KindText, Required: true},
			{Key: "severity", Name: "Severity", Kind: tracker.KindSelect, Default: &severity, Options: []tracker.Option{
				{Key: "low", Name: "Low"},
				{Key: "medium", Name: "Medium"},
				{Key: "high", Name: "High"},
			}},
			{Key: "resolution", Name: "Resolution", Kind: tracker.KindMarkdown},
		},
		Statuses: []tracker.Status{
			{Key: "open", Name: "Open", Category: tracker.CategoryBacklog},
			{Key: "doing", Name: "Doing", Category: tracker.CategoryActive},
			{Key: "fixed", Name: "Fixed", Category: tracker.CategoryDone},
			{Key: "dropped", Name: "Dropped", Category: tracker.CategoryCanceled},
		},
		Initial: "open",
		Transitions: []tracker.Transition{
			{From: "open", To: "doing"},
			{From: "doing", To: "open"},
			{From: "doing", To: "fixed", RequiredFields: []tracker.FieldKey{"resolution"}},
			{From: "fixed", To: "open"},
			{From: "open", To: "dropped"},
			{From: "dropped", To: "open"},
		},
	}
}
