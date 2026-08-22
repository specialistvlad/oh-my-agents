package tracker_test

import "github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"

// Minted-shaped identifiers, as the system stores them (ADR-0009). Named here
// so no test depends on an id's spelling — which is the property ids exist to
// protect.
const (
	typeBug tracker.TypeID = "bug-9c2x"

	fieldSummary    tracker.FieldID = "summary-3k1p"
	fieldSeverity   tracker.FieldID = "severity-8m2q"
	fieldResolution tracker.FieldID = "resolution-5v7d"
	fieldTags       tracker.FieldID = "tags-2c6r"

	statusOpen    tracker.StatusID = "open-1a4f"
	statusDoing   tracker.StatusID = "doing-6b2c"
	statusFixed   tracker.StatusID = "fixed-7d9e"
	statusDropped tracker.StatusID = "dropped-2f8h"

	optionLow    tracker.OptionID = "low-4j6k"
	optionMedium tracker.OptionID = "medium-9l3m"
	optionHigh   tracker.OptionID = "high-5n8p"
	optionUI     tracker.OptionID = "ui-7q1s"
	optionAPI    tracker.OptionID = "api-3t5u"
)

// bugType is a realistic type used across the validation tests: a required
// field, a select with options, a field only a late transition demands, and a
// workflow that is a graph rather than a line.
func bugType() tracker.ItemType {
	severity := tracker.Select(optionMedium)
	return tracker.ItemType{
		ID:   typeBug,
		Name: "Bug",
		Fields: []tracker.FieldDef{
			{ID: fieldSummary, Name: "Summary", Kind: tracker.KindText, Required: true},
			{ID: fieldSeverity, Name: "Severity", Kind: tracker.KindSelect, Default: &severity, Options: []tracker.Option{
				{ID: optionLow, Name: "Low"},
				{ID: optionMedium, Name: "Medium"},
				{ID: optionHigh, Name: "High"},
			}},
			{ID: fieldResolution, Name: "Resolution", Kind: tracker.KindMarkdown},
			{ID: fieldTags, Name: "Tags", Kind: tracker.KindMultiSelect, Options: []tracker.Option{
				{ID: optionUI, Name: "UI"},
				{ID: optionAPI, Name: "API"},
			}},
		},
		Statuses: []tracker.Status{
			{ID: statusOpen, Name: "Open", Category: tracker.CategoryBacklog},
			{ID: statusDoing, Name: "Doing", Category: tracker.CategoryActive},
			{ID: statusFixed, Name: "Fixed", Category: tracker.CategoryDone},
			{ID: statusDropped, Name: "Dropped", Category: tracker.CategoryCanceled},
		},
		Initial: statusOpen,
		Transitions: []tracker.Transition{
			{From: statusOpen, To: statusDoing},
			{From: statusDoing, To: statusOpen},
			{From: statusDoing, To: statusFixed, RequiredFields: []tracker.FieldID{fieldResolution}},
			{From: statusOpen, To: statusDropped},
			{From: statusDoing, To: statusDropped},
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
		Type:   typeBug,
		Title:  "It breaks",
		Status: statusOpen,
		Fields: map[tracker.FieldID]tracker.Value{
			fieldSummary:  tracker.Text("it breaks"),
			fieldSeverity: tracker.Select(optionMedium),
		},
	}
}
