package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

type StepSetResponse struct{}

func (step StepSetResponse) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepSetResponse) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	const location = "render.StepSetResponse.Post"

	transaction := struct {
		URI     string `json:"uri"     form:"uri"`     // The URI of the object being responded to
		Type    string `json:"type"    form:"type"`    // The Response.Type (Like, Dislike, etc)
		Content string `json:"content" form:"content"` // Addional Value (for Emoji, etc)
	}{}

	// Receive the transaction data
	if err := bind(renderer.request(), &transaction); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding transaction"))
	}

	// Retrieve the currently authenticated user
	user, err := renderer.getUser()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting user"))
	}

	// Create a new response object
	responseService := renderer.factory().Response()

	response := model.NewResponse()
	response.UserID = user.UserID
	response.ActorID = user.ProfileURL
	response.ObjectID = transaction.URI
	response.Type = transaction.Type
	response.Content = transaction.Content

	// Save the response to the database
	if err := responseService.SetResponse(&response); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
	}

	// Carry on, carry onnnnn...
	return Continue()
}
