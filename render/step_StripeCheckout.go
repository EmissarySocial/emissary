package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
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
	priceID, _ := stream.Data.GetStringOK("priceId")

	api, err := factory.StripeClient()

	if err != nil {
		return derp.Wrap(err, location, "Error getting Stripe client")
	}

	address := renderer.Host()

	params := &stripe.CheckoutSessionParams{
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			Price:    stripe.String(priceID),
			Quantity: stripe.Int64(1),
		}},
		Mode:             stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:       stripe.String(address + "/" + stream.StreamID.Hex() + "/success?session={CHECKOUT_SESSION_ID}"),
		CancelURL:        stripe.String(address + "/" + stream.ParentID.Hex()),
		CustomerCreation: stripe.String("if_required"),
		PhoneNumberCollection: &stripe.CheckoutSessionPhoneNumberCollectionParams{
			Enabled: stripe.Bool(true),
		},
	}

	// If tax rates are assigned, then add them to the order
	if taxRateID, ok := stream.Data.GetStringOK("taxId"); ok && (taxRateID != "") {
		for index := range params.LineItems {
			params.LineItems[index].TaxRates = stripe.StringSlice([]string{taxRateID})
		}
	} else {
		// Otherwise, use automatic tax calculation
		// TODO: LOW: This could be a parameter :)
		params.AutomaticTax = &stripe.CheckoutSessionAutomaticTaxParams{Enabled: stripe.Bool(true)}
	}

	// If shipping rates are assinged, then add them to the order
	if shippingMethod, ok := stream.Data.GetStringOK("shippingMethod"); ok && (shippingMethod != "") {
		shippingRates := convert.SliceOfString(shippingMethod)
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
