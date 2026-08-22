package trackerhttp

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
)

// parseQuery reads a [tracker.Query] from the URL.
//
// Repeated parameters mean "any of these", matching how the query itself
// treats a list — `?status=todo-0001&status=doing-0001` is the two together
// rather than the second replacing the first.
//
// Custom-field matches are deliberately absent. A field match needs a typed
// value, and there is no honest way to read one from a string without knowing
// the field's kind, so guessing would produce a filter that quietly matches
// nothing. It waits for a caller that needs it.
func parseQuery(r *http.Request) (tracker.Query, error) {
	params := r.URL.Query()
	q := tracker.Query{
		Roots: params.Get("roots") == "true",
		Page:  pageOf(r),
	}
	for _, v := range params["type"] {
		q.Types = append(q.Types, tracker.TypeID(v))
	}
	for _, v := range params["status"] {
		q.Statuses = append(q.Statuses, tracker.StatusID(v))
	}
	for _, v := range params["category"] {
		category := tracker.StatusCategory(v)
		if !known(category) {
			return tracker.Query{}, fmt.Errorf("unknown category %q", v)
		}
		q.Categories = append(q.Categories, category)
	}
	if id := params.Get("parent"); id != "" {
		q.Parent = ptr(tracker.ItemID(id))
	}
	if id := params.Get("subtree"); id != "" {
		q.Subtree = ptr(tracker.ItemID(id))
	}
	if err := parseTime(params, &q); err != nil {
		return tracker.Query{}, err
	}
	sort, err := parseSort(params)
	if err != nil {
		return tracker.Query{}, err
	}
	q.Sort = sort
	return q, nil
}

func parseTime(params url.Values, q *tracker.Query) error {
	raw := params.Get("updated_since")
	if raw == "" {
		return nil
	}
	since, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return fmt.Errorf("updated_since must be an RFC 3339 time: %w", err)
	}
	q.UpdatedSince = &since
	return nil
}

// parseSort refuses an unknown key rather than falling back to the default.
// Silently ignoring it would return a correct-looking page in the wrong order,
// which is worse than saying no.
func parseSort(params url.Values) (tracker.Sort, error) {
	sort := tracker.Sort{By: tracker.SortCreatedAt, Desc: params.Get("desc") == "true"}
	switch by := params.Get("sort"); by {
	case "":
	case string(tracker.SortCreatedAt), string(tracker.SortUpdatedAt),
		string(tracker.SortTitle), string(tracker.SortRank):
		sort.By = tracker.SortKey(by)
	default:
		return tracker.Sort{}, fmt.Errorf("unknown sort %q", by)
	}
	return sort, nil
}

func known(c tracker.StatusCategory) bool {
	switch c {
	case tracker.CategoryBacklog, tracker.CategoryActive, tracker.CategoryBlocked,
		tracker.CategoryDone, tracker.CategoryCanceled:
		return true
	default:
		return false
	}
}

func ptr[T any](v T) *T { return &v }
