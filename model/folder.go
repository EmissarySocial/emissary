package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder is a human-friendly container that can hold zero or more streams
type Folder struct {
	FolderID        primitive.ObjectID `json:"folderId" bson:"_id"`
	ParentID        primitive.ObjectID `json:"parentId" bson:"parentId"`
	Token           string             `json:"token"    bson:"token"`
	Label           string             `json:"label"    bson:"label"`
	Sort            int                `json:"sort"     bson:"sort"` // The sort order of this folder within its container
	SubFolders      []Folder           `json:"-"        bson:"-"`    // An array of sub-folders (populated after being loaded)
	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the unique identifier for this object.
func (folder *Folder) ID() string {
	return folder.FolderID.Hex()
}

// HasParent returns TRUE if this Folder has a valid parentID
func (folder *Folder) HasParent() bool {
	return !folder.ParentID.IsZero()
}
