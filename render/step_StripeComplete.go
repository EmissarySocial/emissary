package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/davecgh/go-spew/spew"
)

// StepStripeComplete represents an action-step that forwards the user to a new page.
type StepStripeComplete struct{}

func (step StepStripeComplete) UseGlobalWrapper() bool {
	return true
}

func (step StepStripeComplete) Get(renderer Renderer, _ io.Writer) error {

	const location = "render.StepStripeComplete.Get"

	factory := renderer.factory()
	api, err := factory.StripeClient()

	if err != nil {
		return derp.Report(derp.Wrap(err, location, "Error getting Stripe client"))
	}

	renderer.SetBool("valid", false) // Set valid=false until everything is loaded.

	// Get session from URL query
	sessionID := renderer.context().QueryParam("session")

	if sessionID == "" {
		renderer.SetString("error", "Session ID is missing")
	}

	// Retrieve Session data from Stripe
	session, err := api.CheckoutSessions.Get(sessionID, nil)

	if err != nil {
		renderer.SetString("error", "Invalid Stripe Session ID: '"+sessionID+"'")
		derp.Report(err)
		return nil
	}

	// Retrieve Customer data from Stripe
	customer, err := api.Customers.Get(session.Customer.ID, nil)

	if err != nil {
		renderer.SetString("error", "Error retrieving customer record: '"+session.Customer.ID+"'")
		derp.Report(err)
		return nil
	}

	renderer.SetInt64("subTotal", session.AmountSubtotal)
	renderer.SetInt64("total", session.AmountSubtotal)
	renderer.SetString("customerId", session.Customer.ID)

	renderer.SetString("customerName", customer.Name)
	renderer.SetString("customerEmail", customer.Email)
	renderer.SetString("customerPhone", customer.Phone)

	renderer.SetBool("valid", true)

	// Success!!

	spew.Dump("StripeComplete: success")
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStripeComplete) Post(renderer Renderer) error {
	return nil
}
