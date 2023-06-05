package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const EncryptionKeyEncodingPlaintext = "plaintext"

type EncryptionKey struct {
	EncryptionKeyID primitive.ObjectID `json:"encryptionKeyId" bson:"_id"`
	UserID          primitive.ObjectID `json:"userId"          bson:"userId"`
	Type            string             `json:"type"            bson:"type"`
	Encoding        string             `json:"encoding"        bson:"encoding"`
	PublicPEM       string             `json:"publicPEM"       bson:"publicPEM"`
	PrivatePEM      string             `json:"privatePEM"      bson:"privatePEM"`

	journal.Journal `json:"journal" bson:",inline"`
}

func NewEncryptionKey() EncryptionKey {
	return EncryptionKey{
		EncryptionKeyID: primitive.NewObjectID(),
	}
}

func EncryptionKeySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"encryptionKeyId": schema.String{Format: "objectId", Required: true},
			"userId":          schema.String{Format: "objectId", Required: true},
			"type":            schema.String{Required: true},
			"encoding":        schema.String{Required: true},
			"publicPEM":       schema.String{Required: true},
			"privatePEM":      schema.String{Required: true},
		},
	}
}

/******************************
 * data.Object Interface
 ******************************/

func (encryptionKey *EncryptionKey) ID() string {
	return encryptionKey.EncryptionKeyID.Hex()
}
