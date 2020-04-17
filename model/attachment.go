package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Attachment represents a file that has been uploaded to the software
type Attachment struct {
	AttachmentID primitive.ObjectID `json:"attachmentId" bson:"_id"`
	Filename     string             `json:"filename"     bson:"filename"`
	Size         int64              `json:"size"         bson:"size"`
	URL          string             `json:"url"          bson:"url"`

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key of this object
func (attachment *Attachment) ID() string {
	return attachment.AttachmentID.Hex()
}
