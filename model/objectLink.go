package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ObjectLink defines a single member of a conversation objectLink.  The actor may be a local or remote user, and the
// Object may be a local stream or an inbox message.
type ObjectLink struct {
	ObjectLinkID primitive.ObjectID `bson:"_id"`        // Unique identifier for this ObjectLink record
	Context      string             `bson:"objectLink"` // ActivityPubURL of the context that this linked object belongs to.
	InReplyTo    string             `bson:"inReplyTo"`  // ActivityPubURL of the resource that the linked object replies to.
	Object       string             `bson:"object"`     // ActivityPubURL of the Object being linked to
	Recipients   sliceof.String     `bson:"recipients"` // ActivityPubURLs of the intended recipients of the linked object

	journal.Journal `json:"-" bson:",inline"`
}

// NewObjectLink returns a fully initialized ObjectLink object
func NewObjectLink() ObjectLink {
	return ObjectLink{
		ObjectLinkID: primitive.NewObjectID(),
		Recipients:   sliceof.String{},
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

// ID returns the unique identifier for this ObjectLink (in string format)
func (objectLink ObjectLink) ID() string {
	return objectLink.ObjectLinkID.Hex()
}

func (objectLink ObjectLink) Fields() []string {
	return []string{"_id", "objectLink", "inReplyTo", "actor", "recipients", "object"}
}
