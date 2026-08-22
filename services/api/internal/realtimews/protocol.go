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
	// KindSet writes a setting.
	KindSet = "set"
	// KindDelete removes one.
	KindDelete = "delete"
	// KindProjectCreate creates a project.
	KindProjectCreate = "project.create"
	// KindProjectRename changes a project's display name.
	KindProjectRename = "project.rename"
	// KindProjectRepoint changes where a project's data lives, moving no
	// files.
	KindProjectRepoint = "project.repoint"
	// KindProjectRemove removes a project and deletes its root directory.
	KindProjectRemove = "project.remove"
	// KindItemCreate creates a tracker item.
	KindItemCreate = "item.create"
	// KindItemUpdate edits one, stating the version it expects.
	KindItemUpdate = "item.update"
	// KindItemDelete removes one, stating the version it expects.
	KindItemDelete = "item.delete"
	// KindCommentAdd posts a comment on an item.
	KindCommentAdd = "comment.add"
	// KindItemReorder moves an item between two neighbors.
	KindItemReorder = "item.reorder"
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

	// Key and Value carry a settings mutation.
	Key   string          `json:"key,omitempty"`
	Value json.RawMessage `json:"value,omitempty"`

	// Project, Name and Root carry a project mutation.
	Project string `json:"project,omitempty"`
	Name    string `json:"name,omitempty"`
	Root    string `json:"root,omitempty"`

	// Item, Version and Body carry a tracker mutation. Version is the
	// compare-and-swap version an edit expects to replace, and is required
	// on anything that changes an existing item.
	Item    string          `json:"item,omitempty"`
	Version int64           `json:"version,omitempty"`
	Body    json.RawMessage `json:"body,omitempty"`

	// Idempotency makes a command safe to send twice.
	//
	// A client that never saw an acknowledgement cannot know whether its
	// command arrived, so after reconnecting its only sensible move is to
	// send it again. Without this the replay applies the write a second
	// time — and for a delete, reports a failure for something that in fact
	// succeeded. Absent means "apply it every time", which is correct for a
	// command the client is willing to repeat.
	Idempotency string `json:"idempotency,omitempty"`
}

// Outbound is a frame to a client.
type Outbound struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
	Room string `json:"room,omitempty"`
	Key  string `json:"key,omitempty"`
	// Status is the code an HTTP client would have seen for the same
	// failure, so both edges describe one failure one way.
	Status int             `json:"status,omitempty"`
	Seq    uint64          `json:"seq,omitempty"`
	Kind   string          `json:"kind,omitempty"`
	Data   json.RawMessage `json:"data,omitempty"`
	Error  string          `json:"error,omitempty"`
	// Result carries what a command produced, when it produced something —
	// the created or changed project, for instance. A removal produces
	// nothing and carries none.
	Result json.RawMessage `json:"result,omitempty"`
}
