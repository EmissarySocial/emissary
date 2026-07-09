package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CollectionItem represents a single item that is part of an ActivityPub "collection"
type CollectionItem struct {
	CollectionItemID primitive.ObjectID `bson:"collectionItemId,omitempty"` // Unique ID of a document in this database
	CollectionID     primitive.ObjectID `bson:"collectionId,omitempty"`     // Unique ID of the parent Collection document in this database
	UserID           primitive.ObjectID `bson:"userId,omitempty"`           // Unique ID of the User who owns the Collection
	ParentID         primitive.ObjectID `bson:"parentId,omitempty"`         // Unique ID of the parent document in this database
	Type             string             `bson:"type,omitempty"`             // Type of collection (Context, Replies, etc.)
	URI              string             `bson:"uri,omitempty"`              // Public URI of the CollectionItem
	InReplyTo        string             `bson:"inReplyTo,omitempty"`        // Public URI of the item this CollectionItem is replying to

	journal.Journal `bson:",inline"` // Embedded journal fields
}

// NewCollectionItem returns a fully initialized CollectionItem
func NewCollectionItem() CollectionItem {
	return CollectionItem{}
}

// ID returns the string version of the CollectionItemID
func (collectionItem CollectionItem) ID() string {
	return collectionItem.CollectionItemID.Hex()
}
