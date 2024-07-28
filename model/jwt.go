package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTKey represents
type JWTKey struct {
	JWTKeyID  primitive.ObjectID `bson:"_id"`       // Unique identifier for this key (used by MongoDB)
	KeyName   string             `bson:"keyName"`   // Name of this key (used by the application)
	Algorithm string             `bson:"algorithm"` // Algorithm used to generate this key (AES)
	Encrypted string             `bson:"encrypted"` // Encrypted value

	journal.Journal `json:"-" bson:",inline"`
}

// NewJWTKey returns a fully initialized JWTKey object
func NewJWTKey() JWTKey {

	return JWTKey{
		JWTKeyID: primitive.NewObjectID(),
	}
}

// ID implements the data.Object interface, and
//
//	returns the unique identifier for this key
func (key JWTKey) ID() string {
	return key.JWTKeyID.Hex()
}
