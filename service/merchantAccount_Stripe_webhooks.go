package service

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// stripe_processWebhook processes product webhook events from Stripe
func (service *MerchantAccount) stripe_processWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) error {

	const location = "service.MerchantAccount.stripe_processWebhook"

	var event stripe.Event

	// Parse the request body into a Stripe event
	switch merchantAccount.LiveMode {

	// In TEST mode, just unmarshal the body directly
	case false:

		if err := json.Unmarshal(body, &event); err != nil {
			return derp.Wrap(err, location, "Error unmarshalling webhook event")
		}

	// In LIVE mode, use the Stripe library to validate event
	default:

		// Retrieve the webhook signing key from the Vault
		vault, err := service.DecryptVault(merchantAccount, "webhookSecret")

		if err != nil {
			return derp.Wrap(err, location, "Error decrypting webhook secret")
		}

		// Parse and validate the Webhook event
		event, err = webhook.ConstructEvent(body, header.Get("Stripe-Signature"), vault.GetString("webhookSecret"))

		if err != nil {
			return derp.Wrap(err, location, "Error parsing webhook event")
		}
	}

	// Filter out other non-subscription events
	if !strings.HasPrefix(string(event.Type), "customer.subscription.") {
		return derp.NotImplementedError(location)
	}

	// Unpack the Subscription from the Webhook event
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return derp.Wrap(err, location, "Error unmarshalling subscription data")
	}

	// Load the Privilege associated with this Stripe Subscription.
	privilege := model.NewPrivilege()
	if err := service.privilegeService.LoadByRemotePurchaseID(subscription.ID, &privilege); err != nil {
		return derp.Wrap(err, location, "Error loading privilege")
	}

	// If the underlying Subscription is no longer active, then remove the Privilege
	if isActive := stripeapi.SubscriptionIsActive(subscription); !isActive {

		if err := service.privilegeService.Delete(&privilege, "Updated via WebHook"); err != nil {
			return derp.Wrap(err, location, "Error syncing privilege records")
		}
	}

	// Successfully processed the WebHook event
	return nil
}
