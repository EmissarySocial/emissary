package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EncryptionKeyEncodingPlaintext marks a key whose PEM values are stored unencrypted
const EncryptionKeyEncodingPlaintext = "plaintext"

// EncryptionKey is a public/private key pair belonging to a User or a Stream
type EncryptionKey struct {
	EncryptionKeyID primitive.ObjectID `json:"encryptionKeyId" bson:"_id"`
	ParentType      string             `json:"parentType"      bson:"parentType"`
	ParentID        primitive.ObjectID `json:"parentId"        bson:"parentId"`
	Encoding        string             `json:"encoding"        bson:"encoding"`
	PublicPEM       string             `json:"publicPEM"       bson:"publicPEM"`
	PrivatePEM      string             `json:"privatePEM"      bson:"privatePEM"`

	journal.Journal `json:"-" bson:",inline"`
}

// NewEncryptionKey returns a fully initialized, empty EncryptionKey
func NewEncryptionKey() EncryptionKey {
	return EncryptionKey{
		EncryptionKeyID: primitive.NewObjectID(),
	}
}

// EncryptionKeySchema returns the rosetta schema that describes a EncryptionKey
func EncryptionKeySchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"encryptionKeyId": schema.String{Format: "objectId", Required: true},
			"parentId":        schema.String{Format: "objectId", Required: true},
			"parentType":      schema.String{Required: true},
			"encoding":        schema.String{Required: true},
			"publicPEM":       schema.String{Required: true},
			"privatePEM":      schema.String{Required: true},
		},
	}
}

/******************************
 * data.Object Interface
 ******************************/

// ID returns the primary key of this EncryptionKey, as a string
func (encryptionKey *EncryptionKey) ID() string {
	return encryptionKey.EncryptionKeyID.Hex()
}
