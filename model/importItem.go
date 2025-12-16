package model

import (
	"strings"

	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ImportItem represents a single record that was/will be imported from a remote source.
type ImportItem struct {
	ImportItemID primitive.ObjectID `bson:"_id"`       // Unique identifier for this ImportItem
	ImportID     primitive.ObjectID `bson:"importId"`  // Import that this ImportItem is a part of
	UserID       primitive.ObjectID `bson:"userId"`    // User who owns this Import and ImportItem
	RemoteID     primitive.ObjectID `bson:"remoteId"`  // Unique identifier of the record in the remote database (if native emissary import)
	LocalID      primitive.ObjectID `bson:"localId"`   // Unique identifier of the record in the local database
	Type         string             `bson:"type"`      // Type of collection that this ImportItem comes from
	ImportURL    string             `bson:"importUrl"` // URL where the original item can be / was imported
	RemoteURL    string             `bson:"remoteUrl"` // Original URL of the item on the remote server
	LocalURL     string             `bson:"localUrl"`  // URL of the itme on the local server
	StateID      string             `bson:"stateId"`   // State of this ImportItem
	Message      string             `bson:"message"`   // Human-friendly message about the state of this ImportItem

	journal.Journal `bson:",inline"`
}

// NewImportItem returns a fully initialized ImportItem object
func NewImportItem() ImportItem {
	return ImportItem{
		ImportItemID: primitive.NewObjectID(),
	}
}

// ID returns the unique identifier of this ImportItem
// This is a part of the data.Object interface
func (item ImportItem) ID() string {
	return item.ImportItemID.Hex()
}

// ReplaceRemoteIDs processes a string, replacing all occurrances of the old RemoteID with the new LocalID
func (item ImportItem) ReplaceRemoteIDs(value *string) bool {

	if remoteIDstring := item.RemoteID.Hex(); strings.Contains(*value, remoteIDstring) {
		localIDstring := item.LocalID.Hex()
		*value = strings.ReplaceAll(*value, remoteIDstring, localIDstring)
		return true
	}

	return false
}
