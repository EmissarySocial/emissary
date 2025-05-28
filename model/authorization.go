package model

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID      primitive.ObjectID   `json:"U,omitzero"` // ID of the signed-in User
	IdentityID  primitive.ObjectID   `json:"I,omitzero"` // ID of the authenticated Identity
	GroupIDs    []primitive.ObjectID `json:"G,omitzero"` // deprecated IDs for all server-level groups that the User belongs to
	ClientID    primitive.ObjectID   `json:"C,omitzero"` // ID of the OAuth Application/Client
	Scope       string               `json:"S,omitzero"` // OAuth Scopes that this user has access to
	DomainOwner bool                 `json:"O,omitzero"` // If TRUE, then this user is an owner of this domain
	APIUser     bool                 `json:"A,omitzero"` // If TRUE, then this user is an API user

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
	return !authorization.UserID.IsZero()
}

// IsIdentity returns TRUE if this authorization is valid and has a non-zero IdentityID
func (authorization Authorization) IsIdentity() bool {
	return !authorization.IdentityID.IsZero()
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
