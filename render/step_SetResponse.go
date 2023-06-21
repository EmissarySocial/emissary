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

	txn := struct {
		Type    string `json:"type"    form:"type"`    // The Response.Type (Like, Dislike, etc)
		Content string `json:"content" form:"content"` // Addional Value (for Emoji, etc)
	}{}

	// Receive the transaction data
	if err := renderer.context().Bind(&txn); err != nil {
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
	response.ObjectID = renderer.Permalink()
	response.Type = txn.Type
	response.Content = txn.Content

	// Save the response to the database
	if err := responseService.SetResponse(&response); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
	}

	return Continue().WithEvent("refreshResponses", response.ObjectID)
	//	TriggerEvent(renderer.context(), `{"refreshResponses":{"url":"`+response.ObjectID+`"}}`)
	// return nil
}
