package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/domain"
	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/stripeapi"
	"github.com/benpate/derp"
	"github.com/benpate/steranko"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/webhook"
)

// PostCheckoutWebhook processes inbound webhook events for a specific MerchantAccount
func PostStripeWebhook_Checkout(ctx *steranko.Context, factory *domain.Factory, merchantAccount *model.MerchantAccount) error {

	const location = "handler.PostStripeWebhook_Checkout"

	// RULE: MerchantAccount must be a STRIPE or STRIPE CONNECT account
	if merchantAccount.Type != model.ConnectionProviderStripe {
		return derp.NotImplementedError(location, "This Webhook handler is only valid for Stripe accounts")
	}

	// Retrieve the Merchant Account from the database
	merchantAccountService := factory.MerchantAccount()

	// Retrieve the webhook signing key from the Vault
	vault, err := merchantAccountService.DecryptVault(merchantAccount, "webhookSecret")

	if err != nil {
		return derp.Wrap(err, location, "Error decrypting webhook secret")
	}

	if err = stripe_ProcessWebhook(factory, ctx.Request(), vault.GetString("webhookSecret"), merchantAccount.LiveMode); err != nil {

		// Suppress errors from subscriptions that are not found on this server
		if derp.IsNotFound(err) {
			return nil
		}

		// All other errors are reported to the caller
		return derp.Wrap(err, location, "Error processing webhook data")
	}

	// Success. WebHook complete.
	return ctx.NoContent(http.StatusOK)
}

// stripe_ProcessWebhook processes product webhook events from Stripe
func stripe_ProcessWebhook(factory *domain.Factory, request *http.Request, webhookSecret string, liveMode bool) error {

	const location = "handler.stripe_ProcessWebhook"

	// Parse the request body into a Stripe event
	event, err := stripe_UnmarshalEvent(request, webhookSecret, liveMode)

	if err != nil {
		return derp.Wrap(err, location, "Error unmarshalling Stripe event")
	}

	// Filter out other non-subscription events
	if !strings.HasPrefix(string(event.Type), "customer.subscription.") {
		return nil
	}

	// Unpack the Subscription from the Webhook event
	var subscription stripe.Subscription
	if err := json.Unmarshal(event.Data.Raw, &subscription); err != nil {
		return derp.Wrap(err, location, "Error unmarshalling subscription data")
	}

	// Load the Privilege associated with this Stripe Subscription.
	privilege := model.NewPrivilege()
	if err := factory.Privilege().LoadByRemotePurchaseID(subscription.ID, &privilege); err != nil {
		return derp.Wrap(err, location, "Error loading privilege")
	}

	// If the underlying Subscription is no longer active, then remove the Privilege
	if isActive := stripeapi.SubscriptionIsActive(subscription); !isActive {

		if err := factory.Privilege().Delete(&privilege, "Updated via WebHook"); err != nil {
			return derp.Wrap(err, location, "Error syncing privilege records")
		}
	}

	// Successfully processed the WebHook event
	return nil
}

func stripe_UnmarshalEvent(request *http.Request, webhookSecret string, liveMode bool) (stripe.Event, error) {

	const location = "service.MerchantAccount.stripe_UnmarshalEvent"

	// Read the request Body as a byte array
	reader := io.LimitReader(request.Body, 65535)
	body, err := io.ReadAll(reader)

	if err != nil {
		return stripe.Event{}, derp.Wrap(err, location, "Error reading request body")
	}

	defer request.Body.Close()

	// For regular LIVE requests, use the Stripe library to validate the event
	if liveMode {

		result, err := webhook.ConstructEvent(body, request.Header.Get("Stripe-Signature"), webhookSecret)

		if err != nil {
			return stripe.Event{}, derp.Wrap(err, location, "Error parsing webhook event")
		}

		return result, nil
	}

	// For testmode requests, just unmarshal the body directly
	result := stripe.Event{}

	if err := json.Unmarshal(body, &result); err != nil {
		return stripe.Event{}, derp.Wrap(err, location, "Error unmarshalling webhook event")
	}

	return result, nil
}
