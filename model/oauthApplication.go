package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OAuthApplication struct {
	OAuthApplicationID primitive.ObjectID `json:"oauthApplicationId" bson:"_id"`
	ApplicationID      string             `json:"applicationId"      bson:"applicationId"`
	Name               string             `json:"name"               bson:"name"`
	Website            string             `json:"website"            bson:"website"`
	RedirectURIs       []string           `json:"redirectUris"       bson:"redirectUris"`
	Scopes             sliceof.String     `json:"scopes"             bson:"scopes"`
	ClientSecret       string             `json:"clientSecret"       bson:"clientSecret"`

	journal.Journal `bson:"journal,inline"`
}

func NewOAuthApplication() OAuthApplication {
	return OAuthApplication{
		OAuthApplicationID: primitive.NewObjectID(),
		RedirectURIs:       make([]string, 0),
		Scopes:             make([]string, 0),
	}
}

func (app OAuthApplication) ID() string {
	return app.OAuthApplicationID.Hex()
}

// ToToot converts this object into a Mastodon-compatible Application object
func (app OAuthApplication) ToToot() object.Application {
	return object.Application{
		Name:         app.Name,
		Website:      app.Website,
		ClientID:     app.OAuthApplicationID.Hex(),
		ClientSecret: app.ClientSecret,
	}
}
