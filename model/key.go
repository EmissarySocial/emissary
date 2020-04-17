package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Key is a domain object representing a "to-do" or checklist item that should be completed
type Key struct {
	KeyID    primitive.ObjectID `json:"taskId"   bson:"_id"`      // Unique ID for this key
	DomainID primitive.ObjectID `json:"domainId" bson:"domainId"` // Unique ID for the domain where this key is located
	RoomID   primitive.ObjectID `json:"roomId"   bson:"roomId"`   // Unique ID for the stream that this key encrypts
	Public   string             `json:"public"   bson:"public"`   // Public keys can be read by (anyone?) with access to the key
	Private  string             `json:"private"  bson:"private"`  // Private keys are encrypted by the client BEFORE being sent to this server.
	Codec    string             `json:"codec"    bson:"codec"`    // Codec defines the kind of key being used.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this object
func (task *Key) ID() string {
	return task.KeyID.Hex()
}
