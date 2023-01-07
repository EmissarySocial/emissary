package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (folder *Folder) GetInt(name string) int {
	switch name {

	case "rank":
		return folder.Rank

	default:
		return 0
	}
}

func (folder *Folder) GetString(name string) string {
	switch name {

	case "folderId":
		return folder.FolderID.Hex()

	case "userId":
		return folder.UserID.Hex()

	case "label":
		return folder.Label

	default:
		return ""
	}
}

func (folder *Folder) SetInt(name string, value int) bool {
	switch name {

	case "rank":
		folder.Rank = value
		return true

	default:
		return false
	}
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
