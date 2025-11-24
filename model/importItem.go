package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ImportItem struct {
	ImportItemID primitive.ObjectID `bson:"_id"`      // Unique identifier for this ImportItem
	ImportID     primitive.ObjectID `bson:"importId"` // Import that this ImportItem is a part of
	UserID       primitive.ObjectID `bson:"userId"`   // User who owns this Import and ImportItem
	RemoteID     primitive.ObjectID `bson:"remoteId"` // Unique identifier of the record in the remote database (if native emissary import)
	LocalID      primitive.ObjectID `bson:"localId"`  // Unique identifier of the record in the local database
	Type         string             `bson:"type"`     // Type of collection that this ImportItem comes from
	URL          string             `bson:"url"`      // URL of the original item being imported
	StateID      string             `bson:"stateId"`  // State of this ImportItem
	Message      string             `bson:"message"`  // Human-friendly message about the state of this ImportItem

	journal.Journal `bson:",inline"`
}

func NewImportItem() ImportItem {
	return ImportItem{
		ImportItemID: primitive.NewObjectID(),
	}
}

func (item ImportItem) ID() string {
	return item.ImportItemID.Hex()
}
