package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

type StepSetResponse struct{}

func (step StepSetResponse) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepSetResponse) UseGlobalWrapper() bool {
	return true
}

func (step StepSetResponse) Post(renderer Renderer, _ io.Writer) error {

	const location = "render.StepSetResponse.Post"

	txn := struct {
		Type    string `json:"type"    form:"type"`    // The Response.Type (Like, Dislike, etc)
		Content string `json:"content" form:"content"` // Addional Value (for Emoji, etc)
	}{}

	// Receive the transaction data
	if err := renderer.context().Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding transaction")
	}

	// Retrieve the currently authenticated user
	user, err := renderer.getUser()

	if err != nil {
		return derp.Wrap(err, location, "Error getting user")
	}

	// Create a new response object
	responseService := renderer.factory().Response()

	response := model.NewResponse()
	response.ActorID = user.ProfileURL
	response.ObjectID = renderer.Permalink()
	response.Type = txn.Type
	response.Content = txn.Content

	// Save the response to the database
	if err := responseService.SetResponse(&response); err != nil {
		return derp.Wrap(err, location, "Error setting response")
	}

	return nil
}
