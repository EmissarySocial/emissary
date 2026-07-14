package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/*
ARCHITECTURE NOTE — CollectionItem vs. Response (full write-up in model/response.go)

A CollectionItem is one entry in a Stream's INBOUND collection. For the response-style
collections (Likes / Dislikes / Shares) it records a reaction WE RECEIVED for OUR OWN
content — one row per actor who reacted. This is the object-side mirror of a Response,
which is the OUTBOUND record of a LOCAL actor reacting to something (see response.go).

CollectionItems also back non-response collections (Context, Replies), so this type is
more general than "a received reaction"; CollectionType distinguishes them.

For a reaction, both a Response (outbound, to publish) and a CollectionItem (inbound, to
link the reaction to the liked item) may exist for the SAME reaction — e.g. when a local
user reacts to their own or another local user's content. See response.go for the full
cross-side model and the current known rough edges (dual item-key schemes, the
Response.Save projection shortcut) that are being redesigned.
*/

// CollectionItem represents a single item that is part of an ActivityPub "collection"
type CollectionItem struct {
	CollectionItemID primitive.ObjectID `bson:"_id"`                      // Unique ID of a document in this database
	CollectionID     primitive.ObjectID `bson:"collectionId,omitempty"`   // Unique ID of the parent Collection document in this database
	UserID           primitive.ObjectID `bson:"userId,omitempty"`         // Unique ID of the User who owns the Collection
	ParentID         primitive.ObjectID `bson:"parentId,omitempty"`       // Unique ID of the parent document in this database
	CollectionType   string             `bson:"collectionType,omitempty"` // Type of collection (Context, Replies, etc.)
	URI              string             `bson:"uri,omitempty"`            // Public URI of the CollectionItem

	journal.Journal `bson:",inline"` // Embedded journal fields
}

// NewCollectionItem returns a fully initialized CollectionItem
func NewCollectionItem() CollectionItem {
	return CollectionItem{
		CollectionItemID: primitive.NewObjectID(),
	}
}

// ID returns the string version of the CollectionItemID
func (collectionItem CollectionItem) ID() string {
	return collectionItem.CollectionItemID.Hex()
}

// ActivityPubURL returns the public URI of this CollectionItem. It satisfies
// ActivityPubURLGetter so items can be served in an ActivityPub collection.
func (collectionItem CollectionItem) ActivityPubURL() string {
	return collectionItem.URI
}
