package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Post is an individual update to a particular stream
type Post struct {
	PostID     primitive.ObjectID `json:"postId"     bson:"_id"`                 // Unique ID of this post (only used internally)
	Stream     string             `json:"stream"     bson:"stream"`              // URL token for the stream that owns this post
	Token      string             `json:"token"      bson:"token"`               // URL token for this post
	Content    []Content          `json:"content"    bson:"content,omitempty"`   // Content object(s) to display for this post
	Encrypted  string             `json:"encrypted"  bson:"encrypted,omitempty"` // Encrypted content to display for this post
	Properties string             `json:"properties" bson:"properties"`          // Post-level data is stored here.  This is opaque to the server, and may be JSON-encoded or encrypted by the client.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (post *Post) ID() string {
	return post.PostID.Hex()
}
