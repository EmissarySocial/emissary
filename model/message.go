package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Message is a Domain Object that represents a message by a Person in a Room
type Message struct {
	MessageID  primitive.ObjectID `json:"messageId"   bson:"_id"`      //
	RoomID     primitive.ObjectID `json:"roomId"      bson:"roomId"`   //
	PersonID   primitive.ObjectID `json:"personId"    bson:"personId"` //
	Type       string             `json:"type"        bson:"type"`     //
	KeyID      primitive.ObjectID `json:"keyId"       bson:"keyId"`    // Encryption key used to encrypt this data.  If empty, then data is plaintext
	Properties string             `json:"text"        bson:"text"`     // Primary contents of the message.  The message itself is opaque to the server, and may be encrypted with an encryption key.

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key of this record
func (message *Message) ID() string {
	return message.MessageID.Hex()
}
