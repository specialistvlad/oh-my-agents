package realtimews

import (
	"context"
	"errors"
	"net/http"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/settings"
)

// Settings is the write surface the socket exposes. Declared here, in the
// consumer, and satisfied by a store that announces its writes — so a
// mutation over the socket reaches other clients exactly as an HTTP one does,
// without this package knowing anything about that.
type Settings interface {
	Set(ctx context.Context, key settings.Key, doc settings.Document) error
	Delete(ctx context.Context, key settings.Key) error
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
	if in.Key == "" {
		return Outbound{Type: KindError, ID: in.ID, Error: in.Type + " needs a key"}
	}
	// A replay is answered with what the original produced, rather than by
	// doing the work again.
	if done, replayed := keys.Recall(in.Idempotency); replayed {
		return replyTo(in, done.Err)
	}

	var err error
	switch in.Type {
	case KindSet:
		err = store.Set(ctx, settings.Key(in.Key), settings.Document(in.Value))
	case KindDelete:
		err = store.Delete(ctx, settings.Key(in.Key))
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
	case errors.Is(err, settings.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, settings.ErrInvalidKey), errors.Is(err, settings.ErrInvalidDocument):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
