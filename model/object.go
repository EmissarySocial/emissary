package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Object represents an unparseable ActivityPub object that is stored in the database.
type Object struct {
	ObjectID    primitive.ObjectID `json:"objectId" bson:"_id"`            // Unique ID of this Object (assigned by the server)
	UserID      primitive.ObjectID `json:"userId" bson:"userId"`           // UserID who created this Object
	Context     string             `json:"context" bson:"context"`         // ActivityPubURL of the context that this object belongs to.
	Permissions sliceof.String     `json:"permissions" bson:"permissions"` // Permissions associated with this Object
	Value       mapof.Any          `json:"value" bson:"value"`             // Value of the object

	journal.Journal `json:"-" bson:",inline"`
}

// NewObject returns a fully initialized, empty Object
func NewObject() Object {
	return Object{
		ObjectID:    primitive.NewObjectID(),
		Permissions: sliceof.NewString(),
		Value:       mapof.NewAny(),
	}
}

// ID is a part of the data.Object interface
// It returns the string version of the Object's ObjectID
func (object Object) ID() string {
	return object.ObjectID.Hex()
}

// GetJSONLD implements the JSONLDGetter interface
// It returns the raw JSON-LD value of this object
func (object Object) GetJSONLD() mapof.Any {
	return object.Value
}
