package tracker_test

import (
	"encoding/json"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// Clearing a field and leaving it alone are different instructions, and both
// have to survive the wire or a patch means something else at the far end.
func TestPatchDistinguishesClearFromUntouched(t *testing.T) {
	var p tracker.Patch
	body := `{"fields":{"summary-3k1p":null}}`
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	v, named := p.Fields["summary-3k1p"]
	if !named {
		t.Fatal("the field was not named at all; a null must mean clear")
	}
	if v != nil {
		t.Errorf("field = %v, want nil meaning clear", v)
	}
	if _, named := p.Fields["severity-8m2q"]; named {
		t.Error("a field absent from the body must stay absent")
	}
}

func TestPatchRoundTrips(t *testing.T) {
	title, status := "renamed", tracker.StatusID("doing-6b2c")
	value := tracker.Text("hello")
	sent := tracker.Patch{
		Title:  &title,
		Status: &status,
		Fields: map[tracker.FieldID]*tracker.Value{"summary-3k1p": &value},
		Author: tracker.ActorRef{Kind: tracker.ActorAgent, ID: "a1"},
	}
	encoded, err := json.Marshal(sent)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var back tracker.Patch
	if err := json.Unmarshal(encoded, &back); err != nil {
		t.Fatalf("Unmarshal %s: %v", encoded, err)
	}
	if back.Title == nil || *back.Title != title {
		t.Errorf("Title = %v, want %q", back.Title, title)
	}
	if back.Status == nil || *back.Status != status {
		t.Errorf("Status = %v, want %q", back.Status, status)
	}
	if got := back.Fields["summary-3k1p"]; got == nil || !got.Equal(value) {
		t.Errorf("field = %v, want %v", got, value)
	}
	if back.Author.ID != "a1" {
		t.Errorf("Author = %+v, want the agent", back.Author)
	}
}
