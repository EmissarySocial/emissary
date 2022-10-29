package external

import (
	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"golang.org/x/oauth2"
)

// https://golangexample.com/go-twitter-rest-and-streaming-api/

const ProviderTypeTwitter = "TWITTER"

type Twitter struct {
	provider config.Provider
}

func NewTwitter(provider config.Provider) Twitter {
	return Twitter{
		provider: provider,
	}
}

/******************************************
 * Manual Adapter Methods
 ******************************************/

func (adapter Twitter) OAuthConfig() oauth2.Config {

	return oauth2.Config{
		ClientID:     adapter.provider.ClientID,
		ClientSecret: adapter.provider.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://twitter.com/i/oauth2/authorize",
			TokenURL:  "https://twitter.com/i/oauth2/access_token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"tweet.read", "follows.read", "mute.read", "like.read", "block.read", "offline.access"},
	}
}

/******************************************
 * Adapter Methods
 ******************************************/

func (adapter Twitter) PollStreams(client model.Client) error {
	return nil
}

func (adapter Twitter) PostStream(client model.Client) error {
	return nil
}

/* Archived..

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
*/
