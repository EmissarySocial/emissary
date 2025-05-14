package model

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID      primitive.ObjectID   `json:"U,omitzero"`   // ID of the signed-in User
	GuestID     primitive.ObjectID   `json:"GID,omitzero"` // ID of the authenticated Guest
	GroupIDs    []primitive.ObjectID `json:"G,omitzero"`   // deprecated IDs for all server-level groups that the User belongs to
	ClientID    primitive.ObjectID   `json:"C,omitzero"`   // ID of the OAuth Application/Client
	Scope       string               `json:"S,omitzero"`   // OAuth Scopes that this user has access to
	DomainOwner bool                 `json:"O,omitzero"`   // If TRUE, then this user is an owner of this domain
	APIUser     bool                 `json:"A,omitzero"`   // If TRUE, then this user is an API user

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
func (authorization Authorization) IsAuthenticated() bool {

	// If your UserID is zero, then NO, you're not authenticated
	return !authorization.UserID.IsZero()
}

// IsGuest returns TRUE if this authorization is valid and has a non-zero GuestID
func (authorization Authorization) IsGuest() bool {
	// If your GuestID is zero, then NO, you're not a guest
	return !authorization.GuestID.IsZero()
}

// IsGroupMember returns TRUE if this authorization has any one of the specified groupID
func (authorization Authorization) IsGroupMember(groupIDs ...primitive.ObjectID) bool {

	// Check to see if the groupID is in the list of GroupIDs
	for _, groupID := range groupIDs {
		for _, authorizedGroupID := range authorization.GroupIDs {
			if authorizedGroupID == groupID {
				return true
			}
		}
	}

	return false
}

// Scopes returns a slice of scopes that this Authorization token is allowed to use.
// This implements the toot.ScopesGetter interface.
func (authorization Authorization) Scopes() []string {
	return strings.Split(authorization.Scope, " ")
}
