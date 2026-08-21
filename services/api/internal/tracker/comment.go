package tracker

import "time"

// Comment is one message on an item. Everyone works here — humans and agents
// post through the same path, and an agent narrating its progress is an
// ordinary comment, not a separate log.
type Comment struct {
	ID     CommentID
	Item   ItemID
	Author ActorRef
	Body   string

	// ReplyTo threads this comment under another on the same item. One
	// level only: a reply cannot itself be replied to.
	ReplyTo *CommentID

	CreatedAt time.Time
	EditedAt  *time.Time
	Version   Version
}

// NewComment is the input to posting a comment.
type NewComment struct {
	Item    ItemID
	Author  ActorRef
	Body    string
	ReplyTo *CommentID
}
