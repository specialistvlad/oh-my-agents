package tracker

import "time"

// Version is an item's revision counter. Every write bumps it, and every
// write states the version it expects, so two agents editing one item cannot
// silently overwrite each other — the loser gets [ErrVersionConflict] and
// re-reads.
type Version int64

// Item is one piece of work.
//
// Type decides everything variable about it: which statuses Status may hold,
// and which keys Fields may carry. An item is meaningless without its
// [ItemType], and the two are always read together.
type Item struct {
	ID     ItemID
	Type   TypeID
	Title  string
	Body   string
	Status StatusID

	// Parent is the item directly above this one, nil at the root of a
	// tree. Depth is unbounded and any type may parent any other; the only
	// structural rules are that the graph stays acyclic and that a parent
	// cannot resolve before its descendants do.
	Parent *ItemID

	// Fields holds values for the keys declared by Type. A key absent here
	// has no value; there is no distinction between absent and null.
	Fields map[FieldID]Value

	CreatedBy ActorRef
	CreatedAt time.Time
	UpdatedBy ActorRef
	UpdatedAt time.Time
	Version   Version
}

// NewItem is the input to creating an item.
//
// Status is absent deliberately: a new item enters its type's
// [ItemType.Initial] status and gets there without consulting the transition
// graph, so creation cannot be used to sidestep a workflow.
type NewItem struct {
	Type   TypeID
	Title  string
	Body   string
	Parent *ItemID
	Fields map[FieldID]Value
	Author ActorRef
}

// Patch is a partial update. A nil pointer means "leave alone", which is
// what separates "clear the title" from "do not touch the title".
type Patch struct {
	Title  *string
	Body   *string
	Status *StatusID

	// Parent reparents the item, moving its whole subtree with it.
	// ClearParent promotes it to a root instead, and the two are mutually
	// exclusive.
	Parent      *ItemID
	ClearParent bool

	// Fields sets the named fields. A nil entry clears that field; keys
	// absent from the map are left as they are.
	Fields map[FieldID]*Value

	Author ActorRef
}
