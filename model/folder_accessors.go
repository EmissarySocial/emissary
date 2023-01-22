package model

import (
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FolderSchema returns a Rosetta Schema for the Folder object
func FolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"folderId": schema.String{Format: "objectId"},
			"userId":   schema.String{Format: "objectId"},
			"label":    schema.String{MaxLength: 100},
			"rank":     schema.Integer{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (folder *Folder) GetInt(name string) (int, bool) {
	switch name {

	case "rank":
		return folder.Rank, true
	}

	return 0, false
}

func (folder *Folder) GetString(name string) (string, bool) {
	switch name {

	case "folderId":
		return folder.FolderID.Hex(), true

	case "userId":
		return folder.UserID.Hex(), true

	case "label":
		return folder.Label, true
	}

	return "", false
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (folder *Folder) SetInt(name string, value int) bool {
	switch name {

	case "rank":
		folder.Rank = value
		return true
	}

	return false
}

func (folder *Folder) SetString(name string, value string) bool {
	switch name {

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			folder.FolderID = objectID
			return true
		}

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			folder.UserID = objectID
			return true
		}

	case "label":
		folder.Label = value
		return true
	}

	return false
}
