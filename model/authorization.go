package model

import (
	"github.com/davecgh/go-spew/spew"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID      primitive.ObjectID   `json:"U"` // Unique identifier of the User
	GroupIDs    []primitive.ObjectID `json:"G"` // IDs for all server-level groups that the User belongs to
	DomainOwner bool                 `json:"D"` // If TRUE, then this user is an owner of this domain

	jwt.StandardClaims // By embedding the "StandardClaims" object, this record can support standard behaviors, like token expieration, etc.
}

// NewAuthorization generates a fully initialized Authorization object.
func NewAuthorization() Authorization {

	result := Authorization{
		UserID:      primitive.NewObjectID(),
		GroupIDs:    []primitive.ObjectID{},
		DomainOwner: false,
	}

	result.StandardClaims = jwt.StandardClaims{}

	return result
}

// IsAuthenticated returns TRUE if this authorization is valid and has a non-zero UserID
func (authorization *Authorization) IsAuthenticated() bool {

	spew.Dump(authorization)

	if authorization.UserID.IsZero() {
		spew.Dump("zero")
		return false
	}

	if authorization.StandardClaims.Valid() != nil {
		spew.Dump("invalid")
		return false
	}

	spew.Dump("true")
	return true
}
