package realtimews

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
)

// Projects is the project lifecycle the socket exposes. Declared here, in the
// consumer, and satisfied by a store that announces its changes — so a
// mutation over the socket reaches other clients exactly as an HTTP one does.
type Projects interface {
	Create(ctx context.Context, n projects.New) (projects.Project, error)
	Rename(ctx context.Context, id projects.ID, name string) (projects.Project, error)
	Repoint(ctx context.Context, id projects.ID, root string) (projects.Project, error)
	Remove(ctx context.Context, id projects.ID) error
}

// mutateProject applies one project command and returns the reply.
//
// It runs through the same store the HTTP surface uses, so neither edge owns a
// rule the other lacks (ADR-0010).
func mutateProject(
	ctx context.Context, in Inbound, store Projects, keys *idempotency.Keys,
) Outbound {
	if store == nil {
		return Outbound{Type: KindError, ID: in.ID, Error: "this server accepts no project changes over the socket"}
	}
	if done, replayed := keys.Recall(in.Idempotency); replayed {
		return answerReplay(in, done)
	}
	p, err := applyProject(ctx, in, store)
	reply := projectReply(in, p, err)
	keys.Remember(in.Idempotency, idempotency.Outcome{Err: err, Result: reply.Result})
	return reply
}

// applyProject dispatches to the store. Removal returns no project, which is
// why the result is a pointer.
func applyProject(ctx context.Context, in Inbound, store Projects) (*projects.Project, error) {
	id := projects.ID(in.Project)
	switch in.Type {
	case KindProjectCreate:
		p, err := store.Create(ctx, projects.New{Name: in.Name, Root: in.Root})
		return &p, err
	case KindProjectRename:
		p, err := store.Rename(ctx, id, in.Name)
		return &p, err
	case KindProjectRepoint:
		p, err := store.Repoint(ctx, id, in.Root)
		return &p, err
	case KindProjectRemove:
		return nil, store.Remove(ctx, id)
	default:
		return nil, errors.New("unknown project command " + in.Type)
	}
}

// answerReplay answers a command that was already applied, with what it
// produced the first time.
func answerReplay(in Inbound, done idempotency.Outcome) Outbound {
	reply := projectReply(in, nil, done.Err)
	reply.Result = done.Result
	return reply
}

func projectReply(in Inbound, p *projects.Project, err error) Outbound {
	if err != nil {
		return Outbound{
			Type:   KindError,
			ID:     in.ID,
			Error:  err.Error(),
			Status: projectStatusOf(err),
		}
	}
	reply := Outbound{Type: KindAck, ID: in.ID}
	if p != nil {
		if body, marshalErr := json.Marshal(p); marshalErr == nil {
			reply.Result = body
		}
	}
	return reply
}

// projectStatusOf maps a failure onto the status an HTTP client would have
// seen for the same thing.
func projectStatusOf(err error) int {
	switch {
	case errors.Is(err, projects.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, projects.ErrInvalidID),
		errors.Is(err, projects.ErrInvalidName),
		errors.Is(err, projects.ErrInvalidRoot):
		return http.StatusBadRequest
	case errors.Is(err, projects.ErrNotAProjectRoot):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
