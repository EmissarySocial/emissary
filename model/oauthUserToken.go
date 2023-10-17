package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserOAuthToken represents an application-specific token that
// a remote API can use to access a user's account on their behalf
type OAuthUserToken struct {
	OAuthUserTokenID   primitive.ObjectID `json:"userOAuthTokenId"   bson:"_id"`
	OAuthApplicationID primitive.ObjectID `json:"oauthApplicationId" bson:"oauthApplicationId"`
	UserID             primitive.ObjectID `json:"userId"             bson:"userId"`
	Token              string             `json:"token"              bson:"token"`
	ClientSecret       string             `json:"clientSecret"       bson:"clientSecret"`
	Scopes             sliceof.String     `json:"scopes"             bson:"scopes"`

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

// https://docs.joinmastodon.org/api/oauth-scopes/#list-of-scopes
// Mastodon scopes are: read, write, follow, push
// eg: read:reports, or write:statuses
