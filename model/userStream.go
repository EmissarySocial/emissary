package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/derp"
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
	return UserStream{}
}

/*******************************************
 * data.Object Interface
 *******************************************/

// ID returns a string representation of the UserStream's unique identifier
func (userStream *UserStream) ID() string {
	return userStream.UserStreamID.Hex()
}

func (userStream *UserStream) GetObjectID(name string) (primitive.ObjectID, error) {
	return primitive.NilObjectID, derp.NewInternalError("model.UserStream.GetObjectID", "Invalid property", name)
}

func (userStream *UserStream) GetString(name string) (string, error) {
	return "", derp.NewInternalError("model.UserStream.GetString", "Invalid property", name)
}

func (userStream *UserStream) GetInt(name string) (int, error) {
	return 0, derp.NewInternalError("model.UserStream.GetInt", "Invalid property", name)
}

func (userStream *UserStream) GetInt64(name string) (int64, error) {
	return 0, derp.NewInternalError("model.UserStream.GetInt64", "Invalid property", name)
}

func (userStream *UserStream) GetBool(name string) (bool, error) {
	return false, derp.NewInternalError("model.UserStream.GetBool", "Invalid property", name)
}
