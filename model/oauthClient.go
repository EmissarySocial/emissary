package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OAuthClient struct {
	ClientID     primitive.ObjectID `json:"clientId"      bson:"_id"`
	ClientSecret string             `json:"clientSecret"  bson:"clientSecret"`
	Name         string             `json:"name"          bson:"name"`
	Website      string             `json:"website"       bson:"website"`
	RedirectURIs []string           `json:"redirectUris"  bson:"redirectUris"`
	Scopes       sliceof.String     `json:"scopes"        bson:"scopes"`

	journal.Journal `json:"-" bson:",inline"`
}

func NewOAuthClient() OAuthClient {
	return OAuthClient{
		ClientID:     primitive.NewObjectID(),
		RedirectURIs: make([]string, 0),
		Scopes:       make([]string, 0),
	}
}

func (app OAuthClient) ID() string {
	return app.ClientID.Hex()
}

// ToToot converts this object into a Mastodon-compatible Application object
func (app OAuthClient) Toot() object.Application {
	return object.Application{
		Name:         app.Name,
		Website:      app.Website,
		ClientID:     app.ClientID.Hex(),
		ClientSecret: app.ClientSecret,
	}
}
