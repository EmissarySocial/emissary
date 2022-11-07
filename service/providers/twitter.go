package providers

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/config"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/davecgh/go-spew/spew"
	"github.com/dghubble/go-twitter/twitter"
	"golang.org/x/oauth2"
)

/******************************************
 * Twitter API Resources
 * https://golangexample.com/go-twitter-rest-and-streaming-api/
 * https://github.com/dghubble/go-twitter
 ******************************************/

// ProviderTypeTwitter is the providerID for Twitter
const ProviderTypeTwitter = "TWITTER"

type Twitter struct {
	config config.Provider
}

func NewTwitter(provider config.Provider) Twitter {
	return Twitter{
		config: provider,
	}
}

/******************************************
 * Setup / Configuration Methods
 ******************************************/

func (provider Twitter) OAuthConfig() oauth2.Config {

	return oauth2.Config{
		ClientID:     provider.config.ClientID,
		ClientSecret: provider.config.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://twitter.com/i/oauth2/authorize",
			TokenURL:  "https://api.twitter.com/2/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
		Scopes: []string{"tweet.read", "users.read", "follows.read", "mute.read", "like.read", "block.read", "bookmark.read", "offline.access"},
	}
}

func (provider Twitter) SettingsForm() form.Form {
	return form.Form{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// AfterConnect is called after a user has successfully connected their Twitter account
func (provider Twitter) AfterConnect(factory Factory, client *model.Client) error {

	twitterClient := provider.getTwitterClient(client.Token)

	// Get the Twitter User ID
	user, _, err := twitterClient.Accounts.VerifyCredentials(&twitter.AccountVerifyParams{})

	if err != nil {
		return derp.Wrap(err, "service.External.Twitter.AfterConnect", "Error getting Twitter user info")
	}

	// Update client data
	client.SetInt64("userId", user.ID)
	client.SetString("username", user.ScreenName)
	return nil
}

// AfterUpdate is called after a user has successfully updated their Twitter connection
func (provider Twitter) AfterUpdate(factory Factory, client *model.Client) error {
	return nil
}

/******************************************
 * Adapter Methods
 ******************************************/

func (provider Twitter) PollStreams(client *model.Client) <-chan model.Stream {

	// Create a channel to return results
	result := make(chan model.Stream)

	// Connect to Twitter
	twitterClient := provider.getTwitterClient(client.Token)

	go func() {

		// Get the Twitter User ID
		userID := client.GetInt64("userId")

		// Get all tweets from the user's timeline
		tweets, _, err := twitterClient.Timelines.UserTimeline(&twitter.UserTimelineParams{
			UserID: userID,
		})

		// Report errors (but we can't return them)
		if err != nil {
			derp.Report(derp.Wrap(err, "service.External.Twitter.PollStreams", "Error getting Twitter user timeline", userID))
		}

		// Pass each tweet into the channel
		for _, tweet := range tweets {
			spew.Dump(tweet)
			result <- tweetAsStream(tweet)
		}
	}()

	// Woot woot!
	return result
}

/******************************************
 * API Methods
 ******************************************/

// getTwitterClient returns a Twitter client for the given token
func (provider Twitter) getTwitterClient(token *oauth2.Token) *twitter.Client {
	config := provider.OAuthConfig()
	httpClient := config.Client(context.Background(), token)
	return twitter.NewClient(httpClient)
}

func tweetAsStream(tweet twitter.Tweet) model.Stream {

	// Get create time
	createDate, err := tweet.CreatedAtTime()

	if err != nil {
		createDate = time.Now()
	}

	// Return stream
	stream := model.NewStream()
	stream.Description = tweet.Text
	stream.Origin = model.OriginLink{
		Source:     model.LinkSourceTwitter,
		URL:        twitterURL(tweet),
		UpdateDate: createDate.Unix(),
	}

	return stream
}

func twitterURL(tweet twitter.Tweet) string {
	return "https://twitter.com/" + tweet.User.ScreenName + "/status/" + tweet.IDStr
}

/* Archived..

func (provider Twitter) ManualConfig() form.Form {

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
