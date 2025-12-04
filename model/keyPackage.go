package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type KeyPackage struct {
	KeyPackageID primitive.ObjectID `bson:"_id"`
	UserID       primitive.ObjectID `bson:"userId"`
	MediaType    string             `bson:"mediaType"`
	Encoding     string             `bson:"encoding"`
	Content      string             `bson:"content"`
	Generator    string             `bson:"generator"`

	journal.Journal `json:"-" bson:",inline"`
}

func NewKeyPackage() KeyPackage {
	return KeyPackage{
		KeyPackageID: primitive.NewObjectID(),
	}
}

/******************************
 * data.Object Interface
 ******************************/

func (keyPackage *KeyPackage) ID() string {
	return keyPackage.KeyPackageID.Hex()
}
