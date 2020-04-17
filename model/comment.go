package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Comment is a person you talk ABOUT, not WITH.
type Comment struct {
	CommentID  primitive.ObjectID `json:"commentId"  bson:"_id"`
	DomainID   primitive.ObjectID `json:"domainId"   bson:"domainId"`
	StreamID   primitive.ObjectID `json:"roomId"     bson:"roomId"`
	Message    string             `json:"message"    bson:"message"`
	CreateByID string             `json:"createById" bson:"createById"`

	journal.Journal `json:"journal" bson:"journal"`
}

// ID returns the primary key for this record
func (comment *Comment) ID() string {
	return comment.CommentID.Hex()
}
