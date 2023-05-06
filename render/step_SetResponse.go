package render

import (
	"io"

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
		Type  string `json:"type"     form:"type"`  // The Response.Type (Like, Dislike, etc)
		Value string `json:"value"    form:"value"` // Addional Value (for Emoji, etc)
	}{}

	// Receive the transaction data
	if err := renderer.context().Bind(&txn); err != nil {
		return derp.Wrap(err, location, "Error binding transaction")
	}

	// Try to Create/Load the Response
	responseService := renderer.factory().Response()

	objectID := renderer.objectID()
	response, err := responseService.LoadOrCreate(renderer.AuthenticatedID(), objectID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading response", objectID)
	}

	// Set new values in the Response
	response.Type = txn.Type
	response.Value = txn.Value
	response.ObjectID = objectID
	// response.Object = renderer.documentLink()

	// Save the Response to the database (response service will automatically publish to ActivityPub and beyond)
	if err := responseService.Save(&response, "Updated by User"); err != nil {
		return derp.Wrap(err, location, "Error saving response", response)
	}

	return nil
}
