package realtimews

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/idempotency"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
)

// writeTimeout bounds a single frame write, so one wedged client cannot hold
// a writer goroutine forever.
const writeTimeout = 10 * time.Second

// Keepalive defaults. A connection whose network died without a close frame
// looks exactly like an idle one — the socket stays open, its goroutines stay
// parked and the hub keeps a room membership for a client that is gone.
// Pinging is the only way to tell the two apart.
const (
	defaultPingEvery   = 30 * time.Second
	defaultPingTimeout = 10 * time.Second
)

// Hub is what this transport needs: somewhere to register a connection.
// Declared here, in the consumer, per ADR-0002.
type Hub interface {
	Connect() *realtime.Conn
}

// Options configure the handler.
type Options struct {
	// Origins are the browser origins allowed to open a socket. Empty
	// rejects every cross-origin request, which is the safe default and
	// wrong for local development, so the caller supplies the dev origin.
	Origins []string

	// PingEvery and PingTimeout detect a connection whose network died
	// without closing. Zero means the defaults; tests shorten them.
	PingEvery   time.Duration
	PingTimeout time.Duration

	// Settings is the settings write surface. Nil serves a socket that
	// accepts no settings writes.
	Settings Settings

	// Projects is the project lifecycle. Nil serves a socket that accepts
	// no project changes.
	Projects Projects

	// Trackers resolves a project into its tracker. Nil serves a socket
	// that accepts no tracker changes.
	Trackers Trackers

	// Replays remembers commands so a reconnecting client can safely send
	// again what it never saw acknowledged. Nil gets a default.
	Replays *idempotency.Keys
}

func (o Options) pingEvery() time.Duration {
	if o.PingEvery > 0 {
		return o.PingEvery
	}
	return defaultPingEvery
}

func (o Options) pingTimeout() time.Duration {
	if o.PingTimeout > 0 {
		return o.PingTimeout
	}
	return defaultPingTimeout
}

// New returns a handler that upgrades to a WebSocket and serves one hub
// connection per socket.
//
// There is no authentication, by design (ADR-0012). A client may join any
// room it names: rooms are addressing, not access control.
func New(h Hub, opts Options) http.Handler {
	if opts.Replays == nil {
		opts.Replays = idempotency.New(idempotency.Options{})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sock, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: opts.Origins,
		})
		if err != nil {
			slog.Warn("websocket upgrade refused", "err", err, "origin", r.Header.Get("Origin"))
			return
		}
		defer func() { _ = sock.CloseNow() }()

		serve(r.Context(), sock, h.Connect(), opts)
	})
}

// serve runs one connection: a reader goroutine turning frames into hub
// calls, and this goroutine writing deliveries out.
func serve(ctx context.Context, sock *websocket.Conn, conn *realtime.Conn, opts Options) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	if err := write(ctx, sock, Outbound{Type: KindWelcome}); err != nil {
		return
	}
	go read(ctx, cancel, sock, conn, opts)
	go keepalive(ctx, cancel, sock, opts)

	for {
		select {
		case <-ctx.Done():
			return
		case d, open := <-conn.Out():
			if !open {
				return
			}
			if err := write(ctx, sock, frameOf(d)); err != nil {
				return
			}
		}
	}
}

// read consumes client frames until the socket dies, canceling the
// connection's context so the writer stops too.
func read(ctx context.Context, cancel context.CancelFunc, sock *websocket.Conn, conn *realtime.Conn, opts Options) {
	defer cancel()
	for {
		var in Inbound
		if err := wsjson.Read(ctx, sock, &in); err != nil {
			if !errors.Is(err, context.Canceled) && websocket.CloseStatus(err) == -1 {
				slog.Debug("websocket read ended", "err", err)
			}
			return
		}
		if err := write(ctx, sock, handle(ctx, in, conn, opts)); err != nil {
			return
		}
	}
}

// keepalive pings until a ping goes unanswered, then cancels the connection.
//
// Without this a client that vanished — a closed laptop, a dropped network —
// holds a socket, two goroutines and its room memberships indefinitely,
// because nothing ever arrives to fail on. The ping is what turns silence
// into an error.
func keepalive(ctx context.Context, cancel context.CancelFunc, sock *websocket.Conn, opts Options) {
	defer cancel()
	ticker := time.NewTicker(opts.pingEvery())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pinged, done := context.WithTimeout(ctx, opts.pingTimeout())
			err := sock.Ping(pinged)
			done()
			if err != nil {
				slog.Debug("websocket ping unanswered, dropping", "err", err)
				return
			}
		}
	}
}

// handle applies one client frame and returns the reply. Every reply echoes
// the command's id, which is what makes concurrent commands distinguishable.
func handle(ctx context.Context, in Inbound, conn *realtime.Conn, opts Options) Outbound {
	switch in.Type {
	case KindJoin:
		if in.Room == "" {
			return Outbound{Type: KindError, ID: in.ID, Error: "join needs a room"}
		}
		conn.Join(bus.Room(in.Room))
		return Outbound{Type: KindAck, ID: in.ID, Room: in.Room}
	case KindLeave:
		if in.Room == "" {
			return Outbound{Type: KindError, ID: in.ID, Error: "leave needs a room"}
		}
		conn.Leave(bus.Room(in.Room))
		return Outbound{Type: KindAck, ID: in.ID, Room: in.Room}
	case KindPing:
		return Outbound{Type: KindPong, ID: in.ID}
	case KindSet, KindDelete:
		return mutate(ctx, in, opts.Settings, opts.Replays)
	case KindProjectCreate, KindProjectRename, KindProjectRepoint, KindProjectRemove:
		return mutateProject(ctx, in, opts.Projects, opts.Replays)
	case KindItemCreate, KindItemUpdate, KindItemDelete, KindCommentAdd:
		return mutateTracker(ctx, in, opts.Trackers, opts.Replays)
	default:
		return Outbound{Type: KindError, ID: in.ID, Error: "unknown frame type " + in.Type}
	}
}

func frameOf(d realtime.Delivery) Outbound {
	if d.Resync {
		return Outbound{Type: KindResync, Room: string(d.Room)}
	}
	return Outbound{
		Type: KindEvent,
		Room: string(d.Room),
		Seq:  d.Seq,
		Kind: d.Kind,
		Data: d.Data,
	}
}

func write(ctx context.Context, sock *websocket.Conn, frame Outbound) error {
	ctx, cancel := context.WithTimeout(ctx, writeTimeout)
	defer cancel()
	return wsjson.Write(ctx, sock, frame)
}
