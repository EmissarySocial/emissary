package handler

import (
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/paypal"
	"github.com/EmissarySocial/emissary/tools/random"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

func GetUserOAuthConnect_PayPal(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetUserOAuthConnect_PayPal"

	// Load the Connection from the database
	connectionService := factory.Connection()
	connection := model.NewConnection()

	if err := connectionService.LoadByProvider(model.ConnectionProviderPayPal, &connection); err != nil {
		return derp.Wrap(err, location, "Error loading Connection")
	}

	referral := mapof.Any{
		"business_entity": mapof.Any{
			"names": []mapof.Any{
				{
					"business_name": user.DisplayName,
					"type":          "DOING_BUSINESS_AS",
				},
			},
			"emails": []mapof.Any{
				{
					"type":  "CUSTOMER_SERVICE",
					"email": user.EmailAddress,
				},
			},
		},
		"tracking_id": user.ID(),
		"products":    []string{"PPCP"}, // PayPal Complete Payments
		"email":       user.EmailAddress,
	}

	result := mapof.Any{}

	liveMode := connection.Data.GetString("liveMode") == "LIVE"
	serverName := paypal.APIHost(liveMode)
	txn := remote.Post(serverName + "/v2/customer/partner-referrals").
		ContentType("application/json").
		With(options.BearerAuth(connection.Token.AccessToken)).
		JSON(referral).
		Result(&result)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending referral to PayPal")
	}

	return nil
}

func GetUserOAuthAuthorization(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.PostUserOAuthAuthorization"

	// Get the OAuth configuration for this provider
	providerID := strings.ToUpper(ctx.Param("provider"))
	connectionService := factory.Connection()
	config, err := connectionService.GetOAuthConfig(providerID)

	if err != nil {
		return derp.Wrap(err, location, "Error getting OAuth config", providerID)
	}

	// Create a random state string to prevent attacks on the OAuth flow
	state, err := random.GenerateString(32)

	if err != nil {
		return derp.Wrap(err, location, "Error generating random state")
	}

	// Save the state string in the user's profile
	user.Data.SetString("oauth.state", state)
	userService := factory.User()
	if err := userService.Save(user, "Updating OAuth state"); err != nil {
		return derp.Wrap(err, location, "Error saving user profile")
	}

	// Redirect to the OAuth provider
	return ctx.Redirect(302, config.AuthCodeURL(state))
}

func GetUserOAuthCallback(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil
}

func PostUserOAuthRevoke(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {
	return nil
}
