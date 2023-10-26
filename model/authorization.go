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
		GroupIDs:    make([]primitive.ObjectID, 0),
		DomainOwner: false,
	}

	result.RegisteredClaims = jwt.RegisteredClaims{}

	return result
}

// IsAuthenticated returns TRUE if this authorization is valid and has a non-zero UserID
func (authorization *Authorization) IsAuthenticated() bool {

	// If we have a null pointer, then NO, you're not authenticated
	if authorization == nil {
		return false
	}

	// If your UserID is zero, then NO, you're not authenticated
	if authorization.UserID.IsZero() {
		return false
	}

	// If your authorization token is not valid (expired, etc), then NO, you're not authenticated
	if authorization.RegisteredClaims.Valid() != nil {
		return false
	}

	// Yes, you're authenticated
	return true
}

// AllGroupIDs returns a slice of groups that this authorization belongs to,
// including the magic "Anonymous", and (if valid) "Authenticated" groups.
func (authorization *Authorization) AllGroupIDs() []primitive.ObjectID {

	result := []primitive.ObjectID{MagicGroupIDAnonymous}

	if authorization.IsAuthenticated() {
		result = append(result, MagicGroupIDAuthenticated, authorization.UserID)
		result = append(result, authorization.GroupIDs...)
	}

	return result
}
