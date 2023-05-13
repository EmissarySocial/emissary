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
		Type  string `json:"type"  form:"type"`  // The Response.Type (Like, Dislike, etc)
		Value string `json:"value" form:"value"` // Addional Value (for Emoji, etc)
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

	if documentLinker, ok := renderer.object().(DocumentLinker); ok {

		responseService := renderer.factory().Response()
		documentLink := documentLinker.DocumentLink()

		if err := responseService.SetResponse(user.PersonLink(), documentLink, txn.Type, txn.Value); err != nil {
			return derp.Wrap(err, location, "Error setting response")
		}
	}

	return nil
}
