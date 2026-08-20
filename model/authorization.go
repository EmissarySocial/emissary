package model

import (
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/tools/id"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Authorization represents the JWT Claims that the server gives to a user when they sign in.
type Authorization struct {
	UserID           primitive.ObjectID `json:"U,omitzero"`  // ID of the signed-in User
	IdentityID       primitive.ObjectID `json:"I,omitzero"`  // ID of the authenticated Identity
	GroupIDs         id.Slice           `json:"G,omitempty"` // deprecated IDs for all server-level groups that the User belongs to
	ClientID         primitive.ObjectID `json:"C,omitzero"`  // ID of the OAuth Application/Client
	Scope            string             `json:"S,omitzero"`  // OAuth Scopes that this user has access to
	DomainOwner      bool               `json:"O,omitzero"`  // If TRUE, then this user is an owner of this domain
	APIUser          bool               `json:"A,omitzero"`  // If TRUE, then this user is an API user
	Masquerade       bool               `json:"M,omitzero"`  // If TRUE, then this user is an administrator of this domain who is masquerading as another user.
	Revalidate       int64              `json:"R,omitzero"`  // Unix epoch (seconds) when this session was last verified. Steranko re-checks the session against the database once this is older than its revalidation window.
	OAuthUserTokenID primitive.ObjectID `json:"K,omitzero"`  // ID of the OAuthUserToken (grant) this access token was issued from. Lets OAuth API routes load the grant record without a persisted access token, and makes grant deletion (revoke) an effective 401.

	jwt.RegisteredClaims // By embedding the "RegisteredClaims" object, this record can support standard behaviors, like token expiration, etc.
}

// NewAuthorization generates a fully initialized Authorization object.
func NewAuthorization() Authorization {

	result := Authorization{
		UserID:           primitive.NilObjectID,
		IdentityID:       primitive.NilObjectID,
		GroupIDs:         id.NewSlice(),
		ClientID:         primitive.NilObjectID,
		Scope:            "",
		DomainOwner:      false,
		APIUser:          false,
		Masquerade:       false,
		Revalidate:       0,
		OAuthUserTokenID: primitive.NilObjectID,
		RegisteredClaims: jwt.RegisteredClaims{},
	}

	return result
}

// GetRevalidationTime implements the steranko.Revalidatable interface. It reports
// when this session was last verified, and whether that timestamp is set. A zero
// value means the session has not opted in to revalidation (e.g. guest/Identity
// sessions), so Steranko leaves it alone.
func (authorization Authorization) GetRevalidationTime() (time.Time, bool) {

	if authorization.Revalidate == 0 {
		return time.Time{}, false
	}

	return time.Unix(authorization.Revalidate, 0), true
}

// CarryForwardSessionState implements the steranko.SessionCarrier interface, preserving the
// security-critical Masquerade flag across a revalidation re-mint.
func (authorization *Authorization) CarryForwardSessionState(previous jwt.Claims) {

	// Revalidation rebuilds this Authorization from the User record, which resets any field
	// NOT stored there. Masquerade must survive, so it is copied forward. Every other field
	// (DomainOwner, GroupIDs, ...) is left to be re-derived, so a demotion takes effect.
	if previousAuthorization, ok := previous.(*Authorization); ok {
		authorization.Masquerade = previousAuthorization.Masquerade
	}
}

// IsAuthenticated returns TRUE if this authorization is valid and has a non-zero UserID
func (authorization Authorization) IsAuthenticated() bool {
	return !authorization.UserID.IsZero()
}

// NotAuthenticated returns TRUE if this authorization is NOT valid and has a zero UserID
func (authorization Authorization) NotAuthenticated() bool {
	return authorization.UserID.IsZero()
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

// IsGroupMember returns TRUE if this authorization contains any one of the specified groupIDs
func (authorization Authorization) IsGroupMember(groupIDs ...primitive.ObjectID) bool {
	return slice.ContainsAny(authorization.AllGroupIDs(), groupIDs...)
}

// Scopes returns a slice of scopes that this Authorization token is allowed to use.
// This implements the toot.ScopesGetter interface.
func (authorization Authorization) Scopes() []string {
	return strings.Split(authorization.Scope, " ")
}

// Debug returns this Authorization as a map, for logging and troubleshooting
func (authorization Authorization) Debug() mapof.Any {

	return mapof.Any{
		"userID":      authorization.UserID,
		"identityID":  authorization.IdentityID,
		"groupIDs":    authorization.GroupIDs,
		"clientID":    authorization.ClientID,
		"scope":       authorization.Scope,
		"domainOwner": authorization.DomainOwner,
		"apiUser":     authorization.APIUser,
	}
}
