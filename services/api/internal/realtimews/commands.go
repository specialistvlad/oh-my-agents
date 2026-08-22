package realtimews

import (
	"context"
	"errors"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// Settings resolves a project into its settings store, the same way the HTTP
// edge does. The socket never holds a store of its own: a mutation names its
// project and gets one already rooted there (ADR-0009), so neither edge can
// reach across projects.
type Settings interface {
	Settings(ctx context.Context, id projects.ID) (settings.Store, error)
}

// mutate applies one command and returns the reply.
//
// The command runs through the same port the HTTP surface uses, so neither
// edge owns a rule the other lacks (ADR-0008). Validation happens once, in
// the store.
func mutate(ctx context.Context, in Inbound, store Settings, keys *idempotency.Keys) Outbound {
	if store == nil {
		return Outbound{Type: KindError, ID: in.ID, Error: "this server accepts no writes over the socket"}
	}
	if in.Project == "" {
		return Outbound{Type: KindError, ID: in.ID, Error: in.Type + " needs a project"}
	}
	if in.Key == "" {
		return Outbound{Type: KindError, ID: in.ID, Error: in.Type + " needs a key"}
	}
	// A replay is answered with what the original produced, rather than by
	// doing the work again.
	if done, replayed := keys.Recall(in.Idempotency); replayed {
		return replyTo(in, done.Err)
	}

	scoped, err := store.Settings(ctx, projects.ID(in.Project))
	if err == nil {
		switch in.Type {
		case KindSet:
			err = scoped.Set(ctx, settings.Key(in.Key), settings.Document(in.Value))
		case KindDelete:
			err = scoped.Delete(ctx, settings.Key(in.Key))
		}
	}
	keys.Remember(in.Idempotency, idempotency.Outcome{Err: err})
	return replyTo(in, err)
}

// replyTo turns an outcome into a frame, echoing the command's id so a client
// with several in flight can tell which reply is which.
func replyTo(in Inbound, err error) Outbound {
	if err == nil {
		return Outbound{Type: KindAck, ID: in.ID, Key: in.Key}
	}
	return Outbound{Type: KindError, ID: in.ID, Key: in.Key, Error: err.Error(), Status: statusOf(err)}
}

// statusOf maps a failure onto the status an HTTP client would have seen, so
// the two edges describe the same failure the same way.
func statusOf(err error) int {
	switch {
	case errors.Is(err, settings.ErrNotFound), errors.Is(err, projects.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, projects.ErrInvalidID):
		return http.StatusBadRequest
	case errors.Is(err, settings.ErrInvalidKey), errors.Is(err, settings.ErrInvalidDocument):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
