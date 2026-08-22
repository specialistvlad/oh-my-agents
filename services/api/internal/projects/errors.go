package projects

import "errors"

// Errors every implementation returns. An adapter translates whatever its
// backend raises into one of these and lets nothing else escape.
var (
	// ErrNotFound is a project that is not in the registry.
	ErrNotFound = errors.New("projects: not found")
	// ErrInvalidID is an id this system could not have minted.
	ErrInvalidID = errors.New("projects: invalid id")
	// ErrInvalidName is a name that is empty or absurdly long.
	ErrInvalidName = errors.New("projects: invalid name")
	// ErrInvalidRoot is a root that cannot be used — unreachable, or one of
	// the paths removal must never be pointed at.
	ErrInvalidRoot = errors.New("projects: invalid root")
	// ErrNotAProjectRoot is a directory that carries no marker, or one
	// naming a different project. Removal refuses it, which is what keeps a
	// mistyped path from becoming a recursive delete.
	ErrNotAProjectRoot = errors.New("projects: not a project root")
)
