package trackerhttp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

func decodeItems(t *testing.T, body []byte) []tracker.Item {
	t.Helper()
	var out struct {
		Items []tracker.Item `json:"items"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
	return out.Items
}

func TestFindFiltersAndSorts(t *testing.T) {
	h, _ := serve(t)
	for _, title := range []string{"c", "a", "b"} {
		if code := do(t, h, http.MethodPost, url("items"), newBug(title)).Code; code != http.StatusCreated {
			t.Fatalf("creating %q failed", title)
		}
	}
	all := decodeItems(t, do(t, h, http.MethodGet, url("items"), "").Body.Bytes())
	if len(all) != 3 {
		t.Fatalf("found %d, want 3", len(all))
	}

	sorted := decodeItems(t, do(t, h, http.MethodGet, url("items")+"?sort=title", "").Body.Bytes())
	if sorted[0].Title != "a" || sorted[2].Title != "c" {
		t.Errorf("sorted = %v, want a b c", titles(sorted))
	}
	desc := decodeItems(t, do(t, h, http.MethodGet, url("items")+"?sort=title&desc=true", "").Body.Bytes())
	if desc[0].Title != "c" {
		t.Errorf("descending = %v, want c first", titles(desc))
	}

	// Category is the axis generic logic asks along, so it has to work
	// without naming a single user-defined status.
	backlog := decodeItems(t, do(t, h, http.MethodGet, url("items")+"?category=backlog", "").Body.Bytes())
	if len(backlog) != 3 {
		t.Errorf("backlog = %d items, want all 3", len(backlog))
	}
	done := decodeItems(t, do(t, h, http.MethodGet, url("items")+"?category=done", "").Body.Bytes())
	if len(done) != 0 {
		t.Errorf("done = %d items, want none", len(done))
	}
}

// An empty result is an array, so a client never has to handle null.
func TestAnEmptyResultIsAnArray(t *testing.T) {
	h, _ := serve(t)
	body := do(t, h, http.MethodGet, url("items"), "").Body.String()
	if body[:11] != `{"items":[]` {
		t.Errorf("body = %s, want an empty array", body)
	}
}

// Ignoring an unknown sort would return a correct-looking page in the wrong
// order, which is worse than saying no.
func TestUnknownFiltersAreRefused(t *testing.T) {
	h, _ := serve(t)
	for name, query := range map[string]string{
		"sort":     "?sort=invented",
		"category": "?category=invented",
		"time":     "?updated_since=yesterday",
	} {
		t.Run(name, func(t *testing.T) {
			if code := do(t, h, http.MethodGet, url("items")+query, "").Code; code != http.StatusBadRequest {
				t.Errorf("GET %s = %d, want 400", query, code)
			}
		})
	}
}

// Repeated parameters mean "any of these", matching how the query treats a
// list — the second must not replace the first.
func TestRepeatedParametersAccumulate(t *testing.T) {
	h, _ := serve(t)
	query := "?category=backlog&category=done"
	if code := do(t, h, http.MethodGet, url("items")+query, "").Code; code != http.StatusOK {
		t.Fatalf("GET = %d, want 200", code)
	}
	if code := do(t, h, http.MethodPost, url("items"), newBug("x")).Code; code != http.StatusCreated {
		t.Fatal("create failed")
	}
	found := decodeItems(t, do(t, h, http.MethodGet, url("items")+query, "").Body.Bytes())
	if len(found) != 1 {
		t.Errorf("found %d, want the backlog item matched by the first of two categories", len(found))
	}
}

func titles(items []tracker.Item) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = item.Title
	}
	return out
}
