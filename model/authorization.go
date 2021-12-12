package model

import (
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID      primitive.ObjectID   `json:"U"` // Unique identifier of the User
	GroupIDs    []primitive.ObjectID `json:"G"` // IDs for all server-level groups that the User belongs to
	DomainOwner bool                 `json:"O"` // If TRUE, then this user is an owner of this domain

	jwt.RegisteredClaims // By embedding the "RegisteredClaims" object, this record can support standard behaviors, like token expiration, etc.
}

// NewAuthorization generates a fully initialized Authorization object.
func NewAuthorization() Authorization {

	result := Authorization{
		UserID:      primitive.NilObjectID,
		GroupIDs:    []primitive.ObjectID{},
		DomainOwner: false,
	}

	result.RegisteredClaims = jwt.RegisteredClaims{}

	return result
}

// IsAuthenticated returns TRUE if this authorization is valid and has a non-zero UserID
func (authorization *Authorization) IsAuthenticated() bool {

	if authorization.UserID.IsZero() {
		return false
	}

	if authorization.RegisteredClaims.Valid() != nil {
		return false
	}

	return true
}
