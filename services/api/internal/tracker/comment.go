package tracker

import "time"

// Comment is one message on an item. Everyone works here — humans and agents
// post through the same path, and an agent narrating its progress is an
// ordinary comment, not a separate log.
type Comment struct {
	ID     CommentID `json:"id"`
	Item   ItemID    `json:"item"`
	Author ActorRef  `json:"author"`
	Body   string    `json:"body"`
	// ReplyTo threads this comment under another on the same item. One
	// level only: a reply cannot itself be replied to.
	ReplyTo   *CommentID `json:"reply_to"`
	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at"`
	Version   Version    `json:"version"`
}

// NewComment is the input to posting a comment.
type NewComment struct {
	Item    ItemID     `json:"item"`
	Author  ActorRef   `json:"author"`
	Body    string     `json:"body"`
	ReplyTo *CommentID `json:"reply_to"`
}
