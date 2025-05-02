package model

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID       primitive.ObjectID   `json:"U"`           // Unique identifier of the User
	GroupIDs     []primitive.ObjectID `json:"G"`           // IDs for all server-level groups that the User belongs to
	ClientID     primitive.ObjectID   `json:"C,omitempty"` // Unique identifier of the OAuth Application/Client
	VisitorEmail string               `json:"V,omitempty"` // Authenticated email address of an external user
	Scope        string               `json:"S,omitempty"` // OAuth Scopes that this user has access to
	DomainOwner  bool                 `json:"O,omitempty"` // If TRUE, then this user is an owner of this domain
	APIUser      bool                 `json:"A,omitempty"` // If TRUE, then this user is an API user

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

// Scopes returns a slice of scopes that this Authorization token is allowed to use.
// This implements the toot.ScopesGetter interface.
func (authorization Authorization) Scopes() []string {
	return strings.Split(authorization.Scope, " ")
}
