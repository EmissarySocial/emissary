package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTKey represents
type JWTKey struct {
	JWTKeyID       primitive.ObjectID `bson:"_id"`
	KeyName        string             `bson:"keyName"`
	Algorithm      string             `bson:"algorithm"`
	Plaintext      []byte             `bson:"-"`
	EncryptedValue []byte             `bson:"encryptedValue"`

	journal.Journal
}

func NewJWTKey() JWTKey {

	return JWTKey{
		JWTKeyID:       primitive.NewObjectID(),
		Plaintext:      make([]byte, 128),
		EncryptedValue: make([]byte, 128),
	}
}

func (key JWTKey) ID() string {
	return key.JWTKeyID.Hex()
}
