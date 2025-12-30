package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MLSMessage represents a message that was received via the MLS protocol.
// These messages are opaque to the server and are simply stored and forwarded
// to MLS clients as requested.
type MLSMessage struct {
	MLSMessageID primitive.ObjectID `bson:"_id"`     // Unique identifier for this MLSMessage
	UserID       primitive.ObjectID `bson:"userID"`  // The user that this message belongs to
	Type         string             `bson:"type"`    // The type of MLS message (GroupInfo, PublicMessage, PrivateMessage, Welcome)
	Content      string             `bson:"content"` // The base64-encoded content of the MLS message

	journal.Journal `bson:",inline"`
}

// NewMLSMessage returns a fully initialized MLSMessage with a unique ID
func NewMLSMessage() MLSMessage {
	return MLSMessage{
		MLSMessageID: primitive.NewObjectID(),
	}
}

// ID returns the string version of the MLSMessage's unique identifier
func (m MLSMessage) ID() string {
	return m.MLSMessageID.Hex()
}
