package handler

import (
	"net/http"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/paypal"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/remote/options"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/steranko"
)

// GetPayPalConnect initiates the PayPal connection process for a User.
func GetPayPalConnect(ctx *steranko.Context, factory *domain.Factory, user *model.User) error {

	const location = "handler.GetPayPalConnect"

	// Load the Connection from the database
	connectionService := factory.Connection()
	connection := model.NewConnection()

	if err := connectionService.LoadByProvider(model.ConnectionProviderPayPal, &connection); err != nil {
		return derp.Wrap(err, location, "Error loading PayPal Connection")
	}

	// Get the Access Token for this Connection
	token, err := connectionService.GetAccessToken(&connection)

	if err != nil {
		return derp.Wrap(err, location, "Error loading Access Token")
	}

	// Create a MerchantAccount for this User
	merchantAccount := model.NewMerchantAccount()
	merchantAccount.UserID = user.UserID
	merchantAccount.Type = model.ConnectionProviderPayPal
	merchantAccount.LiveMode = connection.Data.GetString("liveMode") == "LIVE"
	merchantAccount.Name = "PayPal Account"

	// Create a new Partner Referral
	// https://developer.paypal.com/docs/api/partner-referrals/v2/
	referral := mapof.Any{
		"tracking_id": merchantAccount.ID(),
		"operations": []mapof.Any{
			{
				"operation": "API_INTEGRATION",
				"api_integration_preference": mapof.Any{
					"rest_api_integration": mapof.Any{
						"integration_method": "PAYPAL",
						"integration_type":   "THIRD_PARTY",
						"third_party_details": mapof.Any{
							"features": []string{"PAYMENT", "PARTNER_FEE", "ACCESS_MERCHANT_INFORMATION"},
						},
					},
				},
			},
		},
		"partner_config_override": mapof.Any{
			"return_url": factory.Host() + "/@me/settings/payments",
		},
		"legal_consents": []mapof.Any{
			{
				"type":    "SHARE_DATA_CONSENT",
				"granted": true,
			},
		},
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
		"products": []string{"PPCP"}, // PayPal Complete Payments
		"email":    user.EmailAddress,
	}

	liveMode := connection.Data.GetString("liveMode") == "LIVE"
	serverName := paypal.APIHost(liveMode)
	result := mapof.Any{}
	txn := remote.Post(serverName + "/v2/customer/partner-referrals").
		ContentType("application/json").
		With(options.BearerAuth(token.AccessToken)).
		With(options.Debug()).
		JSON(referral).
		Result(&result)

	if err := txn.Send(); err != nil {
		return derp.Wrap(err, location, "Error sending referral to PayPal", derp.WithCode(http.StatusInternalServerError))
	}

	// Find the "action_url" in the response
	// https://developer.paypal.com/docs/api/partner-referrals/v2/
	actionURL := ""
	for _, link := range result.GetSliceOfMap("links") {
		if link.GetString("rel") == "action_url" {
			actionURL = link.GetString("href")
			break
		}
	}

	if actionURL == "" {
		return derp.InternalError(location, "PayPal referral action URL not found", result)
	}

	// Save the MerchantAccount
	if err := factory.MerchantAccount().Save(&merchantAccount, "Linked by User"); err != nil {
		return derp.Wrap(err, location, "Error creating MerchantAccount")
	}

	// Forward the User to PayPal to complete the connection.
	return ctx.Redirect(http.StatusFound, actionURL)
}

// PostPayPalWebhook receives and processes PayPal webhook events.
func PostPayPalWebhook(ctx *steranko.Context, factory *domain.Factory) error {

	const location = "handler.PostPayPalWebhook"

	// Collect the request body into a map
	event := mapof.NewAny()
	if err := ctx.Bind(&event); err != nil {
		return derp.Wrap(err, location, "Error unmarshalling webhook event")
	}

	switch event.GetString("event_type") {

	case "MERCHANT.ONBOARDING.COMPLETED":
		return postPayPalWebhook_MerchantOnboardingCompleted(factory, event)
	}

	return derp.NotImplementedError(location, "PayPal webhook event not implemented", event)
}

func postPayPalWebhook_MerchantOnboardingCompleted(factory *domain.Factory, event mapof.Any) error {

	const location = "handler.PostPaypalWebhook_MerchantOnboardingCompleted"

	// Collect values from the webhook event
	resource := event.GetMap("resource")
	partnerClientID := resource.GetString("partner_client_id")
	merchantID := resource.GetString("merchant_id")
	trackingID := resource.GetString("tracking_id")

	// Retrieve the MerchantAccount
	merchantAccount := model.NewMerchantAccount()
	if err := factory.MerchantAccount().LoadByToken(trackingID, &merchantAccount); err != nil {
		return derp.Wrap(err, location, "Error loading MerchantAccount")
	}

	// Update the MerchantAccount with the new information
	merchantAccount.Plaintext.SetString("partnerClientId", partnerClientID)
	merchantAccount.Plaintext.SetString("merchantId", merchantID)

	if err := factory.MerchantAccount().Save(&merchantAccount, "Onboarding Completed"); err != nil {
		return derp.Wrap(err, location, "Error saving MerchantAccount")
	}

	return nil
}
