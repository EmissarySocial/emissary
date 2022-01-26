package model

import (
	"github.com/benpate/data/journal"
	good "github.com/benpate/datatype"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboxItem represents a single ActivityPub activity that exists in a User's inbox
type InboxItem struct {
	InboxItemID primitive.ObjectID `json:"inboxItemId" bson:"_id"`      // Unique identifier for this inbox item
	UserID      primitive.ObjectID `json:"userId"      bson:"userId"`   // ID of the user that owns this item
	Type        string             `json:"type"        bson:"type"`     // The ActivityType for this record
	Original    good.Map           `json:"original"    bson:"original"` // The original data posted with this record.

	journal.Journal `json:"journal" bson:"journal"`
}

// NewInboxItem returns a fully initialized InboxItem
func NewInboxItem() InboxItem {
	return InboxItem{}
}

/**************************
 * data.Object Interface
 **************************/

// ID returns a string representation of this InboxItem's unique identifier
func (item InboxItem) ID() string {
	return item.InboxItemID.Hex()
}

// GetPath implements the path.Getter interface
func (item InboxItem) GetPath(name string) (interface{}, bool) {
	return nil, false
}

// SetPath implements the path.Setter interface
func (item InboxItem) SetPath(name string, value interface{}) error {
	return derp.NewInternalError("whisperverse.model.InboxItem", "SetPath is unimplemented", name, value)
}
