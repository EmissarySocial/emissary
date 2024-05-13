package build

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

type StepSetResponse struct{}

type StepSetResponseTransaction struct {
	URL     string `json:"url"     form:"url"`     // The URL of the object being responded to
	Type    string `json:"type"    form:"type"`    // The Response.Type (Like, Dislike, etc)
	Content string `json:"content" form:"content"` // Addional Value (for Emoji, etc)
	Exists  string `json:"exists"  form:"exists"`  // If TRUE, then create/update the response.  If FALSE, remove it.
}

func (step StepSetResponse) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepSetResponse) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetResponse.Post"

	transaction := StepSetResponseTransaction{}

	// Receive the transaction data
	if err := bind(builder.request(), &transaction); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error binding transaction"))
	}

	// Retrieve the currently authenticated user
	user, err := builder.getUser()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting user"))
	}

	// Set the value in the database
	responseService := builder.factory().Response()

	// Create/Update the response
	if convert.Bool(transaction.Exists) {

		if err := responseService.SetResponse(&user, transaction.URL, transaction.Type, transaction.Content); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
		}

		return Continue()
	}

	// Fall through means DELETE the Response
	if err := responseService.UnsetResponse(&user, transaction.URL, transaction.Type); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
	}

	// Carry on, carry onnnnn...
	return Continue()
}
