package external

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/null"
	"github.com/benpate/rosetta/schema"
)

// https://golangexample.com/go-twitter-rest-and-streaming-api/

const ProviderTypeTwitter = "TWITTER"

type Twitter struct{}

func NewTwitter() Twitter {
	return Twitter{}
}

/******************************************
 * Manual Adapter Methods
 ******************************************/

func (adapter Twitter) ManualConfig() form.Form {

	return form.Form{
		Schema: schema.Schema{
			Element: schema.Object{
				Properties: schema.ElementMap{
					"active": schema.Boolean{},
					"data": schema.Object{
						Properties: schema.ElementMap{
							"accessToken":       schema.String{MaxLength: null.NewInt(100), Required: true},
							"accessTokenSecret": schema.String{MaxLength: null.NewInt(100), Required: true},
						},
					},
				},
			},
		},
		Element: form.Element{
			Type:        "layout-vertical",
			Label:       "Twitter Setup",
			Description: "Sign into your Stripe account and create an API key.  Then, paste the API key into the field below.",
			Children: []form.Element{
				{
					Type:  "text",
					Path:  "data.accessToken",
					Label: "Access Token",
				},
				{
					Type:  "text",
					Path:  "data.accessTokenSecret",
					Label: "Access Token Secret",
				},
				{
					Type:  "toggle",
					Path:  "active",
					Label: "Enable?",
				},
			},
		},
	}
}

/* OAuth Methods

func (adapter Twitter) OAuthConfig() oauth2.Config {

	return oauth2.Config{
		ClientID:     adapter.configuration.ClientID,
		ClientSecret: adapter.configuration.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://twitter.com/i/oauth2/authorize",
			TokenURL:  "https://twitter.com/i/oauth2/access_token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"tweet.read", "follows.read", "mute.read", "like.read", "block.read", "offline.access"},
	}
}

*/

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Twitter) PollStreams(client model.Client) error {
	return nil
}

func (adapter Twitter) PostStream(client model.Client) error {
	return nil
}
