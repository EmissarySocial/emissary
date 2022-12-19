package model

import "go.mongodb.org/mongo-driver/bson/primitive"

func (folder *Folder) GetInt(name string) int {
	switch name {
	case "rank":
		return folder.Rank
	}
	return 0
}

func (folder *Folder) GetObjectID(name string) primitive.ObjectID {
	switch name {
	case "folderId":
		return folder.FolderID
	case "userId":
		return folder.UserID
	}
	return primitive.NilObjectID
}

func (folder *Folder) GetString(name string) string {
	switch name {
	case "label":
		return folder.Label
	}
	return ""
}
