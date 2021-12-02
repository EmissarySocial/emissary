package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/datatype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Filesystem represents a file that has been uploaded to the software
type Filesystem struct {
	FilesystemID primitive.ObjectID `json:"attachmentId" bson:"_id"`
	Name         string             `json:"name"         bson:"name"`
	ConnectInfo  datatype.Map       `json:"connectInfo"  bson:"connectInfo"`

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key of this object
func (filesystem *Filesystem) ID() string {
	return filesystem.FilesystemID.Hex()
}
