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
	"regexp"
	"strings"
	"time"
)

// ID addresses a project, permanently.
//
// It is a primary key a person can read: a stem taken from the name at
// creation plus a suffix that makes it unique. The stem is frozen and will go
// stale — rename a project and the id keeps the old words — and that is the
// point, because a stem that followed the name would be a name (ADR-0009).
// Nothing may parse an ID or infer anything from one.
type ID string

// idPattern is what a minted ID looks like, and what is accepted back. It is
// deliberately a subset of a settings key segment and of a safe path segment,
// because an ID is used as both.
var idPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// Validate reports whether this could be an ID this system minted.
func (id ID) Validate() error {
	switch {
	case id == "":
		return fmt.Errorf("%w: empty", ErrInvalidID)
	case len(id) > 128:
		return fmt.Errorf("%w: longer than 128 bytes", ErrInvalidID)
	case !idPattern.MatchString(string(id)):
		return fmt.Errorf("%w: %q", ErrInvalidID, string(id))
	default:
		return nil
	}
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

// MintID builds an ID from a name and a nonce.
//
// The stem is the name reduced to something readable in a path and a URL. A
// name with nothing usable in it — punctuation, or a script this reduction
// does not handle — still gets an ID, because the nonce alone is enough to
// address it and refusing the name would be worse.
func MintID(name, nonce string) ID {
	stem := stemOf(name)
	if stem == "" {
		return ID("project-" + nonce)
	}
	return ID(stem + "-" + nonce)
}

// maxStemLen bounds the readable half so an ID stays a manageable path
// segment however long the name was.
const maxStemLen = 40

var notStem = regexp.MustCompile(`[^a-z0-9]+`)

func stemOf(name string) string {
	stem := notStem.ReplaceAllString(strings.ToLower(name), "-")
	stem = strings.Trim(stem, "-")
	if len(stem) > maxStemLen {
		stem = strings.Trim(stem[:maxStemLen], "-")
	}
	return stem
}
