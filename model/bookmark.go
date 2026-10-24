package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Bookmark is a private "save for later" mark that a local User places on a post.  It is never
// published to ActivityPub, and is keyed by the post's own URL so it works for any post the User
// can see: one in their feed, on their own site, or on a remote profile.
type Bookmark struct {
	BookmarkID primitive.ObjectID `bson:"_id"`    // Unique ID for this bookmark
	UserID     primitive.ObjectID `bson:"userId"` // Owner (local User)
	URL        string             `bson:"url"`    // ActivityPub URL of the bookmarked post

	journal.Journal `json:"-" bson:",inline"`
}

// NewBookmark returns a fully initialized Bookmark object
func NewBookmark() Bookmark {
	return Bookmark{
		BookmarkID: primitive.NewObjectID(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns a string representation of the Bookmark's unique id.
func (bookmark Bookmark) ID() string {
	return bookmark.BookmarkID.Hex()
}
