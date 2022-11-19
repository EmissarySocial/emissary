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

	journal.Journal `json:"journal" bson:"journal"`
}

func NewEncryptionKey() EncryptionKey {
	return EncryptionKey{
		EncryptionKeyID: primitive.NewObjectID(),
	}
}

func EncryptionKeySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"encryptionKeyId": schema.String{Format: "objectId"},
			"userId":          schema.String{Format: "objectId"},
			"type":            schema.String{},
			"encoding":        schema.String{},
			"publicPEM":       schema.String{},
			"privatePEM":      schema.String{},
		},
		RequiredProps: []string{"encryptionKeyId", "userId", "type", "encoding", "publicPEM", "privatePEM"},
	}
}

/******************************
 * data.Object Interface
 ******************************/

func (encryptionKey *EncryptionKey) ID() string {
	return encryptionKey.EncryptionKeyID.Hex()
}
