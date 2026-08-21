package tracker_test

import "github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"

// bugType is a realistic type used across the validation tests: a required
// field, a select with options, a field only a late transition demands, and a
// workflow that is a graph rather than a line.
func bugType() tracker.ItemType {
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
			{Key: "tags", Name: "Tags", Kind: tracker.KindMultiSelect, Options: []tracker.Option{
				{Key: "ui", Name: "UI"},
				{Key: "api", Name: "API"},
			}},
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
			{From: "open", To: "dropped"},
			{From: "doing", To: "dropped"},
		},
	}
}

func bugSchema() tracker.Schema {
	return tracker.Schema{Types: []tracker.ItemType{bugType()}}
}

// openBug is a valid stored item of that type.
func openBug() tracker.Item {
	return tracker.Item{
		ID:     "b1",
		Type:   "bug",
		Title:  "It breaks",
		Status: "open",
		Fields: map[tracker.FieldKey]tracker.Value{
			"summary":  tracker.Text("it breaks"),
			"severity": tracker.Select("medium"),
		},
	}
}
