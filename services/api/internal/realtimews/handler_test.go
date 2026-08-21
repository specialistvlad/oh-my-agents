package realtimews_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtime"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/realtimews"
)

const wait = 5 * time.Second

// live starts a hub, a bus and a server, and returns a dialed client.
func live(t *testing.T) (*bus.Memory, func(t *testing.T) *websocket.Conn) {
	t.Helper()
	b := bus.NewMemory()
	h := realtime.New()
	ctx, cancel := context.WithCancel(t.Context())

	pump, err := h.Attach(ctx, b)
	if err != nil {
		t.Fatalf("Attach: %v", err)
	}
	done := make(chan struct{})
	go func() { defer close(done); _ = pump() }()

	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{Origins: []string{"*"}}))
	t.Cleanup(func() {
		srv.Close()
		cancel()
		_ = b.Close()
		<-done
	})

	dial := func(t *testing.T) *websocket.Conn {
		t.Helper()
		sock, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], nil)
		if err != nil {
			t.Fatalf("Dial: %v", err)
		}
		closeBody(resp)
		t.Cleanup(func() { _ = sock.CloseNow() })
		if got := recv(t, sock); got.Type != realtimews.KindWelcome {
			t.Fatalf("first frame = %+v, want welcome", got)
		}
		return sock
	}
	return b, dial
}

// closeBody releases the handshake response. websocket.Dial hands it back
// alongside the socket, and it is a real body that leaks if ignored.
func closeBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
}

func send(t *testing.T, sock *websocket.Conn, in realtimews.Inbound) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	if err := wsjson.Write(ctx, sock, in); err != nil {
		t.Fatalf("write %+v: %v", in, err)
	}
}

func recv(t *testing.T, sock *websocket.Conn) realtimews.Outbound {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), wait)
	defer cancel()
	var out realtimews.Outbound
	if err := wsjson.Read(ctx, sock, &out); err != nil {
		t.Fatalf("read: %v", err)
	}
	return out
}

func TestJoinIsAcknowledgedByID(t *testing.T) {
	_, dial := live(t)
	sock := dial(t)

	send(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "c1", Room: "project:p1"})
	got := recv(t, sock)
	if got.Type != realtimews.KindAck || got.ID != "c1" {
		t.Errorf("reply = %+v, want an ack echoing c1", got)
	}
}

// The whole point of no polling: publish on the server, and it arrives.
func TestEventArrivesWithoutAsking(t *testing.T) {
	b, dial := live(t)
	sock := dial(t)
	send(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "c1", Room: "project:p1"})
	recv(t, sock) // ack

	if err := b.Publish(t.Context(), bus.Message{
		Rooms: []bus.Room{"project:p1"},
		Kind:  "setting.changed",
		Data:  json.RawMessage(`{"key":"agent/model"}`),
	}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	got := recv(t, sock)
	if got.Type != realtimews.KindEvent || got.Kind != "setting.changed" {
		t.Fatalf("frame = %+v, want a setting.changed event", got)
	}
	if got.Room != "project:p1" || got.Seq == 0 {
		t.Errorf("frame = %+v, want the room and a sequence", got)
	}
	if string(got.Data) != `{"key":"agent/model"}` {
		t.Errorf("Data = %s, want the payload as published", got.Data)
	}
}

// Two browser tabs on the same room both see it: this is the multi-client
// requirement, tested rather than assumed.
func TestEveryClientInTheRoomSeesIt(t *testing.T) {
	b, dial := live(t)
	first, second := dial(t), dial(t)
	for _, sock := range []*websocket.Conn{first, second} {
		send(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "c1", Room: "room"})
		recv(t, sock)
	}
	if err := b.Publish(t.Context(), bus.Message{Rooms: []bus.Room{"room"}, Kind: "shared"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	for i, sock := range []*websocket.Conn{first, second} {
		if got := recv(t, sock); got.Kind != "shared" {
			t.Errorf("client %d got %+v", i, got)
		}
	}
}

func TestLeaveStopsDelivery(t *testing.T) {
	b, dial := live(t)
	sock := dial(t)
	send(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "1", Room: "room"})
	recv(t, sock)
	send(t, sock, realtimews.Inbound{Type: realtimews.KindLeave, ID: "2", Room: "room"})
	recv(t, sock)

	if err := b.Publish(t.Context(), bus.Message{Rooms: []bus.Room{"room"}, Kind: "ignored"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	// A ping must come back before the event that should never arrive.
	send(t, sock, realtimews.Inbound{Type: realtimews.KindPing, ID: "3"})
	if got := recv(t, sock); got.Type != realtimews.KindPong {
		t.Errorf("frame = %+v, want a pong and no event", got)
	}
}

func TestPingAndBadFrames(t *testing.T) {
	_, dial := live(t)
	sock := dial(t)

	send(t, sock, realtimews.Inbound{Type: realtimews.KindPing, ID: "p"})
	if got := recv(t, sock); got.Type != realtimews.KindPong || got.ID != "p" {
		t.Errorf("ping reply = %+v, want a pong echoing p", got)
	}
	send(t, sock, realtimews.Inbound{Type: "nonsense", ID: "x"})
	if got := recv(t, sock); got.Type != realtimews.KindError || got.ID != "x" {
		t.Errorf("unknown frame reply = %+v, want an error echoing x", got)
	}
	send(t, sock, realtimews.Inbound{Type: realtimews.KindJoin, ID: "y"})
	if got := recv(t, sock); got.Type != realtimews.KindError {
		t.Errorf("join with no room = %+v, want an error", got)
	}
}

func TestOriginIsEnforced(t *testing.T) {
	h := realtime.New()
	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{Origins: []string{"example.com"}}))
	t.Cleanup(srv.Close)

	_, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://evil.test"}},
	})
	closeBody(resp)
	if err == nil {
		t.Error("a socket from a disallowed origin was accepted")
	}
}

func TestClosingRemovesTheConnection(t *testing.T) {
	h := realtime.New()
	srv := httptest.NewServer(realtimews.New(h, realtimews.Options{Origins: []string{"*"}}))
	t.Cleanup(srv.Close)

	sock, resp, err := websocket.Dial(t.Context(), "ws"+srv.URL[len("http"):], nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	closeBody(resp)
	deadline := time.Now().Add(wait)
	for h.Connections() == 0 && time.Now().Before(deadline) {
	}
	if h.Connections() != 1 {
		t.Fatalf("Connections = %d, want 1", h.Connections())
	}
	_ = sock.CloseNow()

	for h.Connections() > 0 && time.Now().Before(deadline) {
	}
	if h.Connections() != 0 {
		t.Errorf("Connections = %d after the client left, want 0", h.Connections())
	}
}
