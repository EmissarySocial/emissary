package render

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// StepEditSubscription is an action that can edit a subscription for the current user.
type StepEditSubscription struct {
}

func (step StepEditSubscription) Get(renderer Renderer, buffer io.Writer) error {

	// Requre that users are signed in to use this modal
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError("render.StepAddSubscription", "User is not authenticated", nil)
	}

	// Get the request context
	context := renderer.context()

	// Load the existing subscription
	subscriptionService := renderer.factory().Subscription()
	subscription := model.NewSubscription()

	if err := subscriptionService.LoadByToken(renderer.AuthenticatedID(), context.QueryParam("subscriptionId"), &subscription); err != nil {
		return derp.Wrap(err, "render.StepEditSubscription", "Error loading subscription")
	}

	// Create a new form
	form := step.getForm(renderer)
	html, err := form.Editor(&subscription, nil)

	if err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error creating form editor", nil)
	}

	// Wrap the form as a modal dialog (with submit buttons)
	html = WrapModalForm(context.Response(), renderer.URL(), html)

	// Done.
	return context.HTML(http.StatusOK, html)

}

func (step StepEditSubscription) UseGlobalWrapper() bool {
	return false
}

func (step StepEditSubscription) Post(renderer Renderer) error {

	var transaction subscription_transaction

	// Requre that users are signed in to use this modal
	if !renderer.IsAuthenticated() {
		return derp.NewUnauthorizedError("render.StepAddSubscription", "User is not authenticated", nil)
	}

	// Collect data from the form POST
	context := renderer.context()

	if err := context.Bind(&transaction); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error reading form data", nil)
	}

	if err := step.getForm(renderer).Schema.Validate(transaction); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Subscription Data is invalid", transaction)
	}

	// Load the existing subscription
	subscriptionService := renderer.factory().Subscription()
	subscription := model.NewSubscription()

	if err := subscriptionService.LoadByToken(renderer.AuthenticatedID(), context.QueryParam("subscriptionId"), &subscription); err != nil {
		return derp.Wrap(err, "render.StepEditSubscription", "Error loading subscription")
	}

	subscription.URL = transaction.URL
	subscription.FolderID = transaction.FolderID
	subscription.PollDuration = transaction.PollDuration
	subscription.PurgeDuration = transaction.PurgeDuration

	// Save the subscription to the database
	if err := subscriptionService.Save(&subscription, "Updated by User"); err != nil {
		return derp.Wrap(err, "render.StepAddSubscription", "Error saving subscription", subscription)
	}

	// Close the Modal Dialog and return
	CloseModal(context, "")
	return context.NoContent(http.StatusOK)
}

func (step StepEditSubscription) getForm(renderer Renderer) form.Form {

	result := subscription_getForm(renderer)
	result.Element.Label = "Edit Subscription"
	return result
}
