package model

import (
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID        primitive.ObjectID   `json:"U"`           // Unique identifier of the User
	GroupIDs      []primitive.ObjectID `json:"G"`           // IDs for all server-level groups that the User belongs to
	ApplicationID primitive.ObjectID   `json:"C,omitempty"` // Unique identifier of the OAuth Application/Client
	DomainOwner   bool                 `json:"O,omitempty"` // If TRUE, then this user is an owner of this domain

	jwt.RegisteredClaims // By embedding the "RegisteredClaims" object, this record can support standard behaviors, like token expiration, etc.
}

// NewAuthorization generates a fully initialized Authorization object.
func NewAuthorization() Authorization {

	result := Authorization{
		UserID:      primitive.NilObjectID,
		GroupIDs:    make([]primitive.ObjectID, 0),
		DomainOwner: false,
	}

	result.RegisteredClaims = jwt.RegisteredClaims{}

	return result
}

// IsAuthenticated returns TRUE if this authorization is valid and has a non-zero UserID
func (authorization *Authorization) IsAuthenticated() bool {

	// nolint:gosimple // This is more readable than "return !authorization.UserID.IsZero()"
	if authorization.UserID.IsZero() {
		return false
	}

	return true
}

// AllGroupIDs returns a complete slice of groups that this authorization belongs to,
// including the magic "Everybody", "Self", and (if valid) "Authenticated" groups.
func (authorization *Authorization) AllGroupIDs() []primitive.ObjectID {

	result := append(authorization.GroupIDs, authorization.UserID, MagicGroupIDAnonymous)

	if authorization.IsAuthenticated() {
		result = append(result, MagicGroupIDAuthenticated)
	}

	return result
}
