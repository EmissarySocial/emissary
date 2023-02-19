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
			"layout":   schema.String{MaxLength: 100},
			"filter":   schema.String{MaxLength: 100},
			"icon":     schema.String{MaxLength: 100},
			"rank":     schema.Integer{},
		},
	}
}

/******************************************
 * Getter Interfaces
 ******************************************/

func (folder *Folder) GetIntOK(name string) (int, bool) {
	switch name {

	case "rank":
		return folder.Rank, true
	}

	return 0, false
}

func (folder *Folder) GetStringOK(name string) (string, bool) {
	switch name {

	case "folderId":
		return folder.FolderID.Hex(), true

	case "filter":
		return folder.Filter, true

	case "icon":
		return folder.Icon, true

	case "label":
		return folder.Label, true

	case "layout":
		return folder.Layout, true

	case "userId":
		return folder.UserID.Hex(), true
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

	case "filter":
		folder.Filter = value
		return true

	case "folderId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			folder.FolderID = objectID
			return true
		}

	case "icon":
		folder.Icon = value
		return true

	case "label":
		folder.Label = value
		return true

	case "layout":
		folder.Layout = value
		return true

	case "userId":
		if objectID, err := primitive.ObjectIDFromHex(value); err == nil {
			folder.UserID = objectID
			return true
		}
	}

	return false
}
