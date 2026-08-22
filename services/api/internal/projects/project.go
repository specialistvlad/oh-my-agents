// Package projects is the lifecycle of a project: the thing every other
// thing in this system is scoped to (ADR-0009).
//
// A project is an id, a name and a root directory. The id is minted, readable
// and permanent; the name is display text and may be edited; the root is where
// the project's data lives and may be re-pointed, which moves no files.
//
// The registry is the authority on what exists. A directory with no record is
// not a project, so there is one answer to "what projects are there" rather
// than a directory scan that can disagree with a file.
package projects

import (
	"fmt"
	"strings"
	"time"

	"github.com/specialistvlad/oh-my-agents/services/api/internal/ids"
)

// ID addresses a project, permanently.
//
// It is a primary key a person can read: a stem taken from the name at
// creation plus a suffix that makes it unique. The stem is frozen and will go
// stale — rename a project and the id keeps the old words — and that is the
// point, because a stem that followed the name would be a name (ADR-0009).
// Nothing may parse an ID or infer anything from one.
type ID string

// Validate reports whether this could be an ID this system minted, wrapping
// [ErrInvalidID] with the reason.
func (id ID) Validate() error {
	if err := ids.Validate("project", string(id)); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidID, err)
	}
	return nil
}

// Project is one project's record. It is what the registry stores and what
// every list and event carries.
type Project struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`

	// Root is the absolute directory holding this project's data.
	Root string `json:"root"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// maxNameLen keeps a name to something a person typed rather than pasted.
const maxNameLen = 200

// ValidateName reports whether a name is usable. It is display text, so
// almost anything goes — but not nothing, and not a wall.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)
	switch {
	case trimmed == "":
		return fmt.Errorf("%w: empty", ErrInvalidName)
	case len(trimmed) > maxNameLen:
		return fmt.Errorf("%w: longer than %d bytes", ErrInvalidName, maxNameLen)
	default:
		return nil
	}
}

// MintID builds an ID from a name. The nonce is what makes it unique; the
// stem is what makes it readable (ADR-0009).
func MintID(name, nonce string) ID {
	return ID(ids.Mint(name, "project", nonce))
}
