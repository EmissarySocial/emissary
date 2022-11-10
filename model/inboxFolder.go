package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InboxFolder struct {
	InboxFolderID primitive.ObjectID `json:"inboxFolderId" bson:"_id"`    // Unique ID for this folder
	UserID        primitive.ObjectID `json:"userId"        bson:"userId"` // ID of the User who owns this folder
	Label         string             `json:"label"         bson:"label"`  // Label of the folder
	Rank          string             `json:"rank"          bson:"rank"`   // Sort order of the folder

	journal.Journal `json:"-" bson:"journal"`
}

func NewInboxFolder() InboxFolder {
	return InboxFolder{
		InboxFolderID: primitive.NewObjectID(),
	}
}

func (folder InboxFolder) ID() string {
	return folder.InboxFolderID.Hex()
}
