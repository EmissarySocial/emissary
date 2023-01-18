package model

import "go.mongodb.org/mongo-driver/bson/primitive"

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

	case "userId":
		return folder.UserID.Hex(), true

	case "label":
		return folder.Label, true
	}

	return "", false
}

func (folder *Folder) SetIntOK(name string, value int) bool {
	switch name {

	case "rank":
		folder.Rank = value
		return true
	}

	return false
}

func (folder *Folder) SetStringOK(name string, value string) bool {
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
