package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/stripe/stripe-go/v72"
)

// StepStripeSetup represents an action-step that forwards the user to a new page.
type StepStripeSetup struct{}

func (step StepStripeSetup) UseGlobalWrapper() bool {
	return false
}

func (step StepStripeSetup) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStripeSetup) Post(renderer Renderer) error {

	const location = "render.StepStripeSetup.Post"

	factory := renderer.factory()
	domainRenderer := renderer.(Domain)
	domain := domainRenderer.domain

	api, err := factory.StripeClient()

	if err != nil {
		return derp.Wrap(err, location, "Error getting Stripe client")
	}

	// Verify that webhooks have been set up on this domain
	if domain.Connections.GetString("stripe_webhook_secret") == "" {

		// Configure webhook
		params := stripe.WebhookEndpointParams{
			URL: stripe.String("https://" + factory.Hostname() + "/webhooks/stripe"),
			EnabledEvents: []*string{
				stripe.String("checkout.session.completed"),
			},
		}

		// Create webhook
		webhook, err := api.WebhookEndpoints.New(&params)

		if err != nil {
			return derp.Wrap(err, location, "Error creating endpoint")
		}

		// Mark webhook as installed
		domain.Connections.SetString("stripe_webhook_secret", webhook.Secret)
	}

	return nil
}
