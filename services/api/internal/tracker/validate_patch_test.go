package tracker_test

import (
	"errors"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func status(k tracker.StatusID) *tracker.StatusID { return &k }

func value(v tracker.Value) *tracker.Value { return &v }

func TestPatchAllowsADeclaredTransition(t *testing.T) {
	if err := bugSchema().ValidatePatch(openBug(), tracker.Patch{Status: status(statusDoing)}); err != nil {
		t.Errorf("open -> doing: %v", err)
	}
}

func TestPatchRefusesAnUndeclaredTransition(t *testing.T) {
	// open -> fixed is not in the graph; open -> doing -> fixed is.
	err := bugSchema().ValidatePatch(openBug(), tracker.Patch{Status: status(statusFixed)})
	if !errors.Is(err, tracker.ErrTransitionNotAllowed) {
		t.Errorf("open -> fixed = %v, want ErrTransitionNotAllowed", err)
	}
}

// A transition's required fields are judged against the item as it will be
// after the patch, so setting the field and moving in one write is allowed.
func TestTransitionRequirementsSeeTheOutcome(t *testing.T) {
	s := bugSchema()
	doing := openBug()
	doing.Status = statusDoing

	if err := s.ValidatePatch(doing, tracker.Patch{Status: status(statusFixed)}); !errors.Is(err, tracker.ErrFieldRequired) {
		t.Errorf("moving to fixed without a resolution = %v, want ErrFieldRequired", err)
	}

	both := tracker.Patch{
		Status: status(statusFixed),
		Fields: map[tracker.FieldID]*tracker.Value{fieldResolution: value(tracker.Markdown("fixed it"))},
	}
	if err := s.ValidatePatch(doing, both); err != nil {
		t.Errorf("setting the resolution and moving together: %v", err)
	}
}

func TestPatchRefusesClearingARequiredField(t *testing.T) {
	p := tracker.Patch{Fields: map[tracker.FieldID]*tracker.Value{fieldSummary: nil}}
	if err := bugSchema().ValidatePatch(openBug(), p); !errors.Is(err, tracker.ErrFieldRequired) {
		t.Errorf("clearing summary = %v, want ErrFieldRequired", err)
	}
}

func TestPatchRefusesSettingAndClearingTheParent(t *testing.T) {
	parent := tracker.ItemID("p1")
	p := tracker.Patch{Parent: &parent, ClearParent: true}
	if err := bugSchema().ValidatePatch(openBug(), p); err == nil {
		t.Error("ValidatePatch accepted a patch that both sets and clears the parent")
	}
}

func TestPatchRefusesUndeclaredFields(t *testing.T) {
	p := tracker.Patch{Fields: map[tracker.FieldID]*tracker.Value{"invented": value(tracker.Text("x"))}}
	if err := bugSchema().ValidatePatch(openBug(), p); !errors.Is(err, tracker.ErrUnknownField) {
		t.Errorf("ValidatePatch = %v, want ErrUnknownField", err)
	}
}

// A patch that does not move the status is not a transition, so a no-op
// status must not be judged against the graph.
func TestRestatingTheCurrentStatusIsNotATransition(t *testing.T) {
	if err := bugSchema().ValidatePatch(openBug(), tracker.Patch{Status: status(statusOpen)}); err != nil {
		t.Errorf("restating the current status: %v", err)
	}
}

// The patch must not mutate the item it is validated against.
func TestValidatePatchLeavesTheItemAlone(t *testing.T) {
	item := openBug()
	p := tracker.Patch{Fields: map[tracker.FieldID]*tracker.Value{fieldResolution: value(tracker.Markdown("x"))}}
	if err := bugSchema().ValidatePatch(item, p); err != nil {
		t.Fatalf("ValidatePatch: %v", err)
	}
	if _, held := item.Fields[fieldResolution]; held {
		t.Error("ValidatePatch wrote through to the item it was checking")
	}
}
