// Package realtimews carries the realtime hub over a WebSocket.
//
// It is the transport and nothing else: it turns frames into calls on a
// [realtime.Conn] and deliveries into frames. Rooms, fan-out and backpressure
// all belong to the hub, so this package can be replaced without any of that
// moving.
package realtimews

import "encoding/json"

// Frame kinds a client sends.
const (
	// KindJoin subscribes the connection to a room.
	KindJoin = "join"
	// KindLeave unsubscribes it.
	KindLeave = "leave"
	// KindPing asks for a pong, so a client can tell a live connection from
	// one that is merely open.
	KindPing = "ping"
)

// Frame kinds the server sends.
const (
	// KindAck confirms a command, echoing its id.
	KindAck = "ack"
	// KindError rejects a command, echoing its id.
	KindError = "error"
	// KindEvent carries something that happened.
	KindEvent = "event"
	// KindResync means the connection missed messages. The client refetches
	// the current state of what it is showing; nothing is replayed. It is
	// the only recovery path in this design.
	KindResync = "resync"
	// KindPong answers a ping.
	KindPong = "pong"
	// KindWelcome is sent once on connect, so a client knows the socket is
	// usable rather than merely accepted.
	KindWelcome = "welcome"
)

// Inbound is a frame from a client.
//
// ID is the client's own correlation value, echoed on the reply. ADR-0008
// requires it because a bidirectional socket has no request/response pairing
// of its own: without an id, a client with two commands in flight cannot tell
// which reply belongs to which.
type Inbound struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Room string `json:"room,omitempty"`
}

// Outbound is a frame to a client.
type Outbound struct {
	Type  string          `json:"type"`
	ID    string          `json:"id,omitempty"`
	Room  string          `json:"room,omitempty"`
	Seq   uint64          `json:"seq,omitempty"`
	Kind  string          `json:"kind,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}
