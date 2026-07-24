package model

import (
	"github.com/benpate/data/journal"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SigninAttempt logs a failed signin attempt for a specific username.
type SigninAttempt struct {
	SigninAttemptID primitive.ObjectID `bson:"_id"`
	Username        string             `bson:"username"`  // Username that was used in the signin attempt
	IPAddress       string             `bson:"ipAddress"` // Resolved client IP that made the attempt (forensics; the lockout window counts by username across all IPs)
	UserAgent       string             `bson:"userAgent"` // User-Agent header of the attempt (forensics only)
	journal.Journal `bson:",inline"`   // Embedded journal fields for tracking creation and updates
}

// NewSigninAttempt returns a SigninAttempt for the given username, recording the
// resolved client IP and User-Agent of the request that made it.
func NewSigninAttempt(username string, ipAddress string, userAgent string) SigninAttempt {
	return SigninAttempt{
		SigninAttemptID: primitive.NewObjectID(),
		Username:        username,
		IPAddress:       ipAddress,
		UserAgent:       userAgent,
	}
}

func (signinAttempt SigninAttempt) ID() string {
	return signinAttempt.SigninAttemptID.Hex()
}
