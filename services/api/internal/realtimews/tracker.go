package realtimews

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/tracker"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/trackerhttp"
)

// Trackers resolves a project into its tracker, the same way the HTTP edge
// does. The socket holds no store of its own: a mutation names its project
// and gets one already rooted there (ADR-0009).
type Trackers interface {
	Tracker(ctx context.Context, id projects.ID) (tracker.Store, error)
}

// mutateTracker applies one tracker command and returns the reply.
//
// It runs through the same store the HTTP surface uses and maps failures with
// the same function, so one failure is described one way whichever edge
// produced it.
func mutateTracker(
	ctx context.Context, in Inbound, scopes Trackers, keys *idempotency.Keys,
) Outbound {
	if scopes == nil {
		return Outbound{Type: KindError, ID: in.ID, Error: "this server accepts no tracker changes over the socket"}
	}
	if in.Project == "" {
		return Outbound{Type: KindError, ID: in.ID, Error: in.Type + " needs a project"}
	}
	if done, replayed := keys.Recall(in.Idempotency); replayed {
		reply := trackerReply(in, nil, done.Err)
		reply.Result = done.Result
		return reply
	}
	result, err := applyTracker(ctx, in, scopes)
	reply := trackerReply(in, result, err)
	keys.Remember(in.Idempotency, idempotency.Outcome{Err: err, Result: reply.Result})
	return reply
}

// applyTracker dispatches to the store. A delete produces nothing, which is
// why the result is an any rather than an item.
func applyTracker(ctx context.Context, in Inbound, scopes Trackers) (any, error) {
	store, err := scopes.Tracker(ctx, projects.ID(in.Project))
	if err != nil {
		return nil, err
	}
	switch in.Type {
	case KindItemCreate:
		var n tracker.NewItem
		if err := json.Unmarshal(in.Body, &n); err != nil {
			return nil, errors.New("malformed item")
		}
		return store.CreateItem(ctx, n)
	case KindItemUpdate:
		var p tracker.Patch
		if err := json.Unmarshal(in.Body, &p); err != nil {
			return nil, errors.New("malformed patch")
		}
		return store.UpdateItem(ctx, tracker.ItemID(in.Item), tracker.Version(in.Version), p)
	case KindItemDelete:
		var by struct {
			Author tracker.ActorRef `json:"author"`
		}
		_ = json.Unmarshal(in.Body, &by) // an absent body means an unattributed delete
		return nil, store.DeleteItem(ctx, tracker.ItemID(in.Item), tracker.Version(in.Version), by.Author)
	case KindItemReorder:
		var at struct {
			After  *tracker.ItemID `json:"after,omitempty"`
			Before *tracker.ItemID `json:"before,omitempty"`
		}
		_ = json.Unmarshal(in.Body, &at) // an absent body means the start
		return nil, store.Reorder(ctx, tracker.ItemID(in.Item), at.After, at.Before)
	case KindCommentAdd:
		var n tracker.NewComment
		if err := json.Unmarshal(in.Body, &n); err != nil {
			return nil, errors.New("malformed comment")
		}
		n.Item = tracker.ItemID(in.Item)
		return store.AddComment(ctx, n)
	default:
		return nil, errors.New("unknown tracker command " + in.Type)
	}
}

func trackerReply(in Inbound, result any, err error) Outbound {
	if err != nil {
		return Outbound{
			Type:   KindError,
			ID:     in.ID,
			Error:  err.Error(),
			Status: trackerhttp.StatusOf(err),
		}
	}
	reply := Outbound{Type: KindAck, ID: in.ID}
	if result != nil {
		if body, marshalErr := json.Marshal(result); marshalErr == nil {
			reply.Result = body
		}
	}
	return reply
}
