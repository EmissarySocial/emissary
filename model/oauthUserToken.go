package model

import (
	"strings"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"github.com/davecgh/go-spew/spew"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserOAuthToken represents an application-specific token that
// a remote API can use to access a user's account on their behalf
type OAuthUserToken struct {
	OAuthUserTokenID primitive.ObjectID `json:"-" bson:"_id"`
	ClientID         primitive.ObjectID `json:"C" bson:"clientId"`
	UserID           primitive.ObjectID `json:"U" bson:"userId"`
	Token            string             `json:"T" bson:"token"`
	APIUser          bool               `json:"A" bson:"apiUser"`
	Scopes           sliceof.String     `json:"S" bson:"scopes"`

	journal.Journal `json:"-" bson:",inline"`
}

// NewOAuthUserToken returns a fully initialized OAuthUserToken
func NewOAuthUserToken() OAuthUserToken {
	return OAuthUserToken{
		OAuthUserTokenID: primitive.NewObjectID(),
		Scopes:           make([]string, 0),
	}
}

/******************************************
 * data.Object Interface
 ******************************************/

func (token OAuthUserToken) ID() string {
	return token.OAuthUserTokenID.Hex()
}

/******************************************
 * AccessLister Interface
 ******************************************/

// State returns the current state of this Stream.
// It is part of the AccessLister interface
func (token OAuthUserToken) State() string {
	return "default"
}

// IsAuthor returns TRUE if the provided UserID the author of this Stream
// It is part of the AccessLister interface
func (token OAuthUserToken) IsAuthor(_ primitive.ObjectID) bool {
	return false
}

// IsMyself returns TRUE if this object directly represents the provided UserID
// It is part of the AccessLister interface
func (token OAuthUserToken) IsMyself(userID primitive.ObjectID) bool {
	spew.Dump("isMyself", userID, token)
	return userID == token.UserID
}

// RolesToGroupIDs returns a slice of Group IDs that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (token OAuthUserToken) RolesToGroupIDs(roles ...string) Permissions {

	spew.Dump("RolesToGroupIDs", roles, defaultRolesToGroupIDs(token.UserID, roles...))
	return defaultRolesToGroupIDs(token.UserID, roles...)
}

// RolesToPrivilegeIDs returns a slice of Privileges that grant access to any of the requested roles.
// It is part of the AccessLister interface
func (token OAuthUserToken) RolesToPrivilegeIDs(roleIDs ...string) Permissions {
	return NewPermissions()
}

/******************************************
 * Other Methods
 ******************************************/

// Code returns the OAuth2 code that is used to request an access token.
// This is just the string version of the ID.
func (token OAuthUserToken) Code() string {
	return token.OAuthUserTokenID.Hex()
}

// JSONResponse returns the token as a map suitable for JSON API responses.
func (token OAuthUserToken) JSONResponse() map[string]any {

	return map[string]any{
		"access_token": token.Token,
		"token_type":   "Bearer",
		"scope":        strings.Join(token.Scopes, " "),
		"created_at":   time.Now().Unix(),
	}
}

// Toot returns the token as a Toot ActivityPub object.Token.
func (token OAuthUserToken) Toot() object.Token {
	return object.Token{
		AccessToken: token.Token,
		TokenType:   "Bearer",
		Scope:       strings.Join(token.Scopes, " "),
		CreatedAt:   time.Now().Unix(),
	}
}
