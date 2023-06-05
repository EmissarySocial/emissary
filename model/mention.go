package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Mention represents a single hyperlink from an external source to an internal object.
// Mentions are created by WebMentions or by ActivityPub "Mention" records
type Mention struct {
	MentionID primitive.ObjectID `json:"mentionId" bson:"_id"`      // Unique ID for this record
	ObjectID  primitive.ObjectID `json:"objectId"  bson:"objectId"` // Unique ID of the internal object that was mentioned
	Type      string             `json:"type"      bson:"type"`     // Type of object that was mentioned (Stream, User)
	StateID   string             `json:"stateId"   bson:"stateId"`  // State of this mention (Validated, Pending, Invalid)
	Origin    OriginLink         `json:"origin"    bson:"origin"`   // Origin information of the site that mentions this object
	Author    PersonLink         `json:"author"    bson:"author"`   // Author information of the person who mentioned this object

	journal.Journal `json:"journal" bson:",inline"`
}

// NewMention returns a fully initialized Mention object
func NewMention() Mention {
	return Mention{
		MentionID: primitive.NewObjectID(),
		Origin:    NewOriginLink(),
		Author:    NewPersonLink(),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns a string representation of the Mention's unique id.
// This method implements the data.Object interface.
func (mention Mention) ID() string {
	return mention.MentionID.Hex()
}
