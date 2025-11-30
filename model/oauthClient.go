package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/rosetta/sliceof"
	"github.com/benpate/toot/object"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OAuthClient struct {
	ClientID     primitive.ObjectID `bson:"_id"`          // Unique identifier for this Client record
	ClientSecret string             `bson:"clientSecret"` // Shared secret used to retrieve OAuth Tokens
	ActorID      string             `bson:"actorId"`      // ActivityPub URL of the actor that created this Client
	Name         string             `bson:"name"`         // Human-friendly name of the Client
	Summary      string             `bson:"summary"`      // Human-friendly summary/description of the Client
	IconURL      string             `bson:"iconUrl"`      // URL of an icon image to display with the Client's name
	Website      string             `bson:"website"`      // Human-friendly website URL for the Client
	RedirectURIs sliceof.String     `bson:"redirectUris"` // Slice of URLs that the Client is allowed to redirect Users to
	Scopes       sliceof.String     `bson:"scopes"`       // OAuth authorization scopes approved for use by this Client

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
