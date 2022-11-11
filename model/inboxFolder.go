package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InboxFolder struct {
	InboxFolderID primitive.ObjectID `json:"inboxFolderId" bson:"_id"`    // Unique ID for this folder
	UserID        primitive.ObjectID `json:"userId"        bson:"userId"` // ID of the User who owns this folder
	Label         string             `json:"label"         bson:"label"`  // Label of the folder
	Rank          int                `json:"rank"          bson:"rank"`   // Sort order of the folder

	journal.Journal `json:"-" bson:"journal"`
}

func NewInboxFolder() InboxFolder {
	return InboxFolder{
		InboxFolderID: primitive.NewObjectID(),
	}
}

func InboxFolderSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"inboxFolderId": schema.String{Format: "objectId"},
			"userId":        schema.String{Format: "objectId"},
			"label":         schema.String{MaxLength: 100},
			"rank":          schema.Integer{},
		},
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

func (folder InboxFolder) ID() string {
	return folder.InboxFolderID.Hex()
}
