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
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
)

// writeTimeout bounds a single frame write, so one wedged client cannot hold
// a writer goroutine forever.
const writeTimeout = 10 * time.Second

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
}

// New returns a handler that upgrades to a WebSocket and serves one hub
// connection per socket.
//
// There is no authentication. A client may join any room it names, so this
// must not be reachable from an untrusted network until auth exists — the
// prerequisite ADR-0008 records.
func New(h Hub, opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sock, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns: opts.Origins,
		})
		if err != nil {
			slog.Warn("websocket upgrade refused", "err", err, "origin", r.Header.Get("Origin"))
			return
		}
		defer func() { _ = sock.CloseNow() }()

		serve(r.Context(), sock, h.Connect())
	})
}

// serve runs one connection: a reader goroutine turning frames into hub
// calls, and this goroutine writing deliveries out.
func serve(ctx context.Context, sock *websocket.Conn, conn *realtime.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer conn.Close()

	if err := write(ctx, sock, Outbound{Type: KindWelcome}); err != nil {
		return
	}
	go read(ctx, cancel, sock, conn)

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
func read(ctx context.Context, cancel context.CancelFunc, sock *websocket.Conn, conn *realtime.Conn) {
	defer cancel()
	for {
		var in Inbound
		if err := wsjson.Read(ctx, sock, &in); err != nil {
			if !errors.Is(err, context.Canceled) && websocket.CloseStatus(err) == -1 {
				slog.Debug("websocket read ended", "err", err)
			}
			return
		}
		if err := write(ctx, sock, handle(in, conn)); err != nil {
			return
		}
	}
}

// handle applies one client frame and returns the reply. Every reply echoes
// the command's id, which is what makes concurrent commands distinguishable.
func handle(in Inbound, conn *realtime.Conn) Outbound {
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
