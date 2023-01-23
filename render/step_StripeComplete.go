package render

import (
	"io"

	"github.com/benpate/derp"
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

	renderer.setBool("valid", false) // Set valid=false until everything is loaded.

	// Get session from URL query
	sessionID := renderer.context().QueryParam("session")

	if sessionID == "" {
		renderer.setString("error", "Session ID is missing")
	}

	// Retrieve Session data from Stripe
	session, err := api.CheckoutSessions.Get(sessionID, nil)

	if err != nil {
		renderer.setString("error", "Invalid Stripe Session ID: '"+sessionID+"'")
		derp.Report(err)
		return nil
	}

	// Retrieve Customer data from Stripe
	customer, err := api.Customers.Get(session.Customer.ID, nil)

	if err != nil {
		renderer.setString("error", "Error retrieving customer record: '"+session.Customer.ID+"'")
		derp.Report(err)
		return nil
	}

	renderer.setInt64("subTotal", session.AmountSubtotal)
	renderer.setInt64("total", session.AmountSubtotal)
	renderer.setString("customerId", session.Customer.ID)

	renderer.setString("customerName", customer.Name)
	renderer.setString("customerEmail", customer.Email)
	renderer.setString("customerPhone", customer.Phone)

	renderer.setBool("valid", true)

	// Success!!
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStripeComplete) Post(renderer Renderer) error {
	return nil
}
