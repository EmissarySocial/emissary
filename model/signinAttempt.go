package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SigninAttempt logs a failed signin attempt for a specific username.
type SigninAttempt struct {
	SigninAttemptID primitive.ObjectID `bson:"_id"`
	Username        string             `bson:"username"` // Username that was used in the signin attempt
	journal.Journal `bson:",inline"`   // Embedded journal fields for tracking creation and updates
}

func NewSigninAttempt(username string) SigninAttempt {
	return SigninAttempt{
		SigninAttemptID: primitive.NewObjectID(),
		Username:        username,
	}
}

func (signinAttempt SigninAttempt) ID() string {
	return signinAttempt.SigninAttemptID.Hex()
}
