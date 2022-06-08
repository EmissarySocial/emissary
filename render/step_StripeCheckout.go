package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/stripe/stripe-go/v72"
)

// StepStripeCheckout represents an action-step that forwards the user to a new page.
type StepStripeCheckout struct{}

func (step StepStripeCheckout) UseGlobalWrapper() bool {
	return false
}

func (step StepStripeCheckout) Get(renderer Renderer, _ io.Writer) error {
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStripeCheckout) Post(renderer Renderer) error {

	const location = "render.StepStripeCheckout.Post"

	factory := renderer.factory()
	streamRenderer := renderer.(*Stream)
	stream := streamRenderer.stream
	priceID := stream.Data.GetString("priceId")

	api, err := factory.StripeClient()

	if err != nil {
		return derp.Wrap(err, location, "Error getting Stripe client")
	}

	address := renderer.Host()

	// Deterimine tax rates (if any)
	taxRates := make([]string, 0)
	if taxrateID := stream.Data.GetString("taxId"); taxrateID != "" {
		taxRates = append(taxRates, taxrateID)
	}

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
			TaxRates: stripe.StringSlice(taxRates),
		}},
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:       stripe.String(address + "/" + stream.StreamID.Hex() + "/success?session={CHECKOUT_SESSION_ID}"),
		CancelURL:        stripe.String(address + "/" + stream.ParentID.Hex()),
		CustomerCreation: stripe.String("if_required"),
	}

	if shippingRates := stream.Data.GetSliceOfString("shippingMethod"); len(shippingRates) > 0 {
		params.ShippingRates = stripe.StringSlice(shippingRates)
		params.ShippingAddressCollection = &stripe.CheckoutSessionShippingAddressCollectionParams{
			AllowedCountries: stripe.StringSlice([]string{"US"}),
		}
	}

	// Create a new session with Stripe checkout.
	s, err := api.CheckoutSessions.New(params)

	if err != nil {
		return derp.Wrap(err, location, "Error creating Stripe Checkout Session")
	}

	// Forward to the Stripe handler
	CloseModal(renderer.context(), s.URL)

	return nil
}
