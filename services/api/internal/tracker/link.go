package tracker

import "time"

// Link is a typed relation between two items that is not containment.
// Parentage lives on [Item.Parent] and forms the tree; links are the graph
// laid over it, and they may cross trees freely.
type Link struct {
	From      ItemID    `json:"from"`
	Kind      LinkKind  `json:"kind"`
	To        ItemID    `json:"to"`
	CreatedBy ActorRef  `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// LinkKind is the meaning of a link, read left to right: From <kind> To.
type LinkKind string

// The link kinds.
const (
	// LinkBlocks means From must resolve before To can proceed. It does not
	// gate resolution the way parentage does — it is advisory.
	LinkBlocks LinkKind = "blocks"
	// LinkDuplicates means From and To describe the same work.
	LinkDuplicates LinkKind = "duplicates"
	// LinkRelates is an untyped association.
	LinkRelates LinkKind = "relates"
	// LinkCauses means From is the cause of To, e.g. a change that
	// introduced a bug.
	LinkCauses LinkKind = "causes"
)
