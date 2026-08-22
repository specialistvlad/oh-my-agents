// Package rooms names the realtime rooms clients subscribe to.
//
// One place, because a room name is a contract between the server that
// publishes to it and the client that joins it, and two spellings of the same
// room is a bug nobody sees until a client is silently deaf.
package rooms

import (
	"github.com/specialistvlad/oh-my-agents/services/api/internal/bus"
	"github.com/specialistvlad/oh-my-agents/services/api/internal/projects"
)

// Projects carries the project list itself: created, renamed, removed.
//
// It sits in the shared scope, above any project, because a client watching
// the list is not yet watching a project — and one that just lost its project
// still needs to hear that it is gone (ADR-0010).
const Projects bus.Room = "projects"

// Project carries everything that happens inside one project (ADR-0009).
func Project(id projects.ID) bus.Room {
	return bus.Room("project:" + string(id))
}
