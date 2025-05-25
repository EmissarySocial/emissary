package service

import (
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/sliceof"
)

// stripe_handleWebhook processes product webhook events from Stripe
func (service *MerchantAccount) stripe_parseCheckoutWebhook(header http.Header, body []byte, merchantAccount *model.MerchantAccount) (sliceof.Object[model.Purchase], error) {

	const location = "service.MerchantAccount.stripe_handleWebhook"

	return nil, derp.NotImplementedError(location, "Stripe Webhook processing is not implemented yet")

	/*
		var event stripe.Event
		var subscription stripe.Subscription

		// Parse the request body into a Stripe event
		switch merchantAccount.LiveMode {

		// In TEST mode, just unmarshal the body directly
		case false:

			if err := json.Unmarshal(body, &event); err != nil {
				return nil, derp.Wrap(err, location, "Error unmarshalling webhook event")
			}

		// In LIVE mode, use the Stripe library to validate event
		default:

			// Retrieve the webhook signing key from the Vault
			vault, err := service.DecryptVault(merchantAccount, "webhookSecret")

			if err != nil {
				return nil, derp.Wrap(err, location, "Error decrypting webhook secret")
			}

			// Parse and validate the Webhook event
			event, err = webhook.ConstructEvent(body, header.Get("Stripe-Signature"), vault.GetString("webhookSecret"))

			if err != nil {
				return nil, derp.Wrap(err, location, "Error parsing webhook event")
			}
		}

		// Filter webhooks for customer.subscription events only
		switch event.Type {
		case "customer.subscription.created":
		case "customer.subscription.updated":
		case "customer.subscription.deleted":
		case "customer.subscription.paused":
		case "customer.subscription.resumed":

		default:
			return nil, derp.NotImplementedError(location, event.Type)
		}

		// Unpack the Product from the Webhook event
		if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
			return nil, derp.Wrap(err, location, "Error unmarshalling product data")
		}

		// Map products from the Webhook into Purchases
		_, purchases, err := service.stripe_mapSubscriptions(merchantAccount, &subscription)

		if err != nil {
			return nil, derp.Wrap(err, location, "Error mapping subscriptions")
		}

		// Success!
		return purchases, nil
	*/
}
