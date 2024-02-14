package model

import (
	"strings"
	"time"

	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
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

	journal.Journal `bson:"journal,inline"`
}

func NewOAuthUserToken() OAuthUserToken {
	return OAuthUserToken{
		OAuthUserTokenID: primitive.NewObjectID(),
		Scopes:           make([]string, 0),
	}
}

func (token OAuthUserToken) ID() string {
	return token.OAuthUserTokenID.Hex()
}

// Code returns the OAuth2 code that is used to request an access token.
// This is just the string version of the ID.
func (token OAuthUserToken) Code() string {
	return token.OAuthUserTokenID.Hex()
}

func (token OAuthUserToken) JSONResponse() map[string]any {

	return map[string]any{
		"access_token": token.Token,
		"token_type":   "Bearer",
		"scope":        strings.Join(token.Scopes, " "),
		"created_at":   time.Now().Unix(),
	}
}

func (token OAuthUserToken) Toot() object.Token {
	return object.Token{
		AccessToken: token.Token,
		TokenType:   "Bearer",
		Scope:       strings.Join(token.Scopes, " "),
		CreatedAt:   time.Now().Unix(),
	}
}
