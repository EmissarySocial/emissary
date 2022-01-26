package model

import (
	"github.com/benpate/data/journal"
	good "github.com/benpate/datatype"
	"github.com/benpate/derp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OutboxItem represents a single ActivityPub activity that exists in a User's inbox
type OutboxItem struct {
	OutboxItemID primitive.ObjectID `json:"inboxItemId" bson:"_id"`      // Unique identifier for this inbox item
	UserID       primitive.ObjectID `json:"userId"      bson:"userId"`   // ID of the user that owns this item
	Type         string             `json:"type"        bson:"type"`     // The ActivityType for this record
	Original     good.Map           `json:"original"    bson:"original"` // The original data posted with this record.

	journal.Journal `json:"journal" bson:"journal"`
}

// NewOutboxItem returns a fully initialized OutboxItem
func NewOutboxItem() OutboxItem {
	return OutboxItem{}
}

/**************************
 * data.Object Interface
 **************************/

// ID returns a string representation of this OutboxItem's unique identifier
func (item OutboxItem) ID() string {
	return item.OutboxItemID.Hex()
}

// GetPath implements the path.Getter interface
func (item OutboxItem) GetPath(name string) (interface{}, bool) {
	return nil, false
}

// SetPath implements the path.Setter interface
func (item OutboxItem) SetPath(name string, value interface{}) error {
	return derp.NewInternalError("whisperverse.model.OutboxItem", "SetPath is unimplemented", name, value)
}
