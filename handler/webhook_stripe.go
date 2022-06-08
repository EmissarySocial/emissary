package handler

import (
	"encoding/json"
	"io/ioutil"

	"github.com/benpate/derp"
	"github.com/labstack/echo/v4"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/webhook"
	"github.com/whisperverse/whisperverse/model"
	"github.com/whisperverse/whisperverse/server"
)

func StripeWebhook(factoryManager *server.Factory) echo.HandlerFunc {

	const location = "handler.StripeWebhook"

	return func(ctx echo.Context) error {

		// Read the request
		body, err := ioutil.ReadAll(ctx.Request().Body)

		if err != nil {
			return derp.Wrap(err, location, "Error reading request body")
		}

		// Get the factory for this domain
		factory, err := factoryManager.ByContext(ctx)

		if err != nil {
			return derp.Wrap(err, location, "Unrecognized domain")
		}

		// Parse the event
		event := stripe.Event{}

		// If we're in test mode, then don't validate Webhook signatures
		if factory.Hostname() == "localhost" {

			if err := json.Unmarshal(body, &event); err != nil {
				return derp.Wrap(err, location, "Error binding request body")
			}
		} else {

			// Get the model.Domain for this domain
			var domain model.Domain
			domainService := factory.Domain()

			if err := domainService.Load(&domain); err != nil {
				return derp.Wrap(err, location, "Error loading domain")
			}

			// Validate the webhook signature
			secret := domain.Connections.GetString("stripe_webhook_secret")

			if secret == "" {
				return derp.NewBadRequestError(location, "Webhooks are not configured on this domain")
			}

			signatureHeader := ctx.Request().Header.Get("Stripe-Signature")

			event, err = webhook.ConstructEvent(body, signatureHeader, secret)

			if err != nil {
				return derp.Wrap(err, location, "Error validating Webhook signature")
			}
		}

		// Verify specific Webhook Types
		if event.Type == "checkout.session.completed" {

			session := &stripe.CheckoutSession{}

			if err := json.Unmarshal(event.Data.Raw, session); err != nil {
				return derp.Wrap(err, location, "Error unmarshalling checkout session data")
			}

			api, err := factory.StripeClient()

			if err != nil {
				return derp.Wrap(err, location, "Error getting Stripe API client")
			}

			// Call the API again to retrieve line items
			params := stripe.CheckoutSessionParams{}
			params.AddExpand("line_items")

			session, err = api.CheckoutSessions.Get(session.ID, &params)

			if err != nil {
				return derp.Wrap(err, location, "Error expanding Webhook line items")
			}

			streamService := factory.Stream()

			// Update inventory for each line item
			for _, lineItem := range session.LineItems.Data {

				var stream model.Stream

				// Load the matching stream
				if err := streamService.LoadByProductID(lineItem.Price.Product.ID, &stream); err != nil {
					return derp.Wrap(err, location, "Error loading Stream by ProductID", lineItem.Product.ID)
				}

				// Check inventory
				if stream.Data.GetBool("trackInventory") {

					quantityOnHand := stream.Data.GetInt("quantityOnHand") - int(lineItem.Quantity)

					stream.Data.SetInt("quantityOnHand", quantityOnHand)

					if quantityOnHand <= 0 {
						stream.StateID = "sold-out"

						// Send realtime update to this stream (and its parent)
						factory.StreamUpdateChannel() <- stream
					}

					if err := streamService.Save(&stream, "webhooks/stripe: updating inventory"); err != nil {
						return derp.Wrap(err, location, "Error updating inventory")
					}
				}
			}
		}

		return nil
	}
}
