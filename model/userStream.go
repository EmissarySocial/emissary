package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserStream tracks a user's last interaction with a Stream.
type UserStream struct {
	UserStreamID primitive.ObjectID `bson:"_id"`
	UserID       primitive.ObjectID `bson:"userId"`
	StreamID     primitive.ObjectID `bson:"streamId"`
	Vote         string             `bson:"vote"`

	journal.Journal `bson:"journal"`
}

// NewUserStream returns a fully initialized UserStream object
func NewUserStream() UserStream {
	return UserStream{
		UserStreamID: primitive.NewObjectID(),
	}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns a string representation of the UserStream's unique identifier
func (userStream *UserStream) ID() string {
	return userStream.UserStreamID.Hex()
}
