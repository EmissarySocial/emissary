package model

import (
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func KeyPackageSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"keyPackageId": schema.String{Format: "objectId", Required: true},
			"userId":       schema.String{Format: "objectId", Required: true},
			"mediaType":    schema.String{Enum: []string{vocab.MediaTypeMLS}, Required: true},
			"encoding":     schema.String{Enum: []string{vocab.EncodingTypeBase64}, Required: true},
			"content":      schema.String{Required: true},
			"generator":    schema.String{Required: true, Format: "url"},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (keyPackage *KeyPackage) GetStringOK(name string) (string, bool) {
	switch name {

	case "keyPackageId":
		return keyPackage.KeyPackageID.Hex(), true

	case "userId":
		return keyPackage.UserID.Hex(), true

	case "mediaType":
		return keyPackage.MediaType, true

	case "encoding":
		return keyPackage.Encoding, true

	case "generator":
		return keyPackage.Generator, true
	}

	return "", false
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (keyPackage *KeyPackage) SetString(name string, value string) bool {
	switch name {

	case "keyPackageId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			keyPackage.KeyPackageID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			keyPackage.UserID = objectID
			return true
		}

	case "mediaType":
		keyPackage.MediaType = value
		return true

	case "encoding":
		keyPackage.Encoding = value
		return true

	case "content":
		keyPackage.Content = value
		return true

	case "generator":
		keyPackage.Generator = value
		return true
	}

	return false
}
