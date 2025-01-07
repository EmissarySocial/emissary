package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

type StepSetResponse struct{}

func (step StepSetResponse) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

func (step StepSetResponse) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetResponse.Post"

	// Receive the transaction data
	transaction := txnStepSetResponse{}

	if err := transaction.Bind(builder.request()); err != nil {
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

type txnStepSetResponse struct {
	URL     string // The URL of the object being responded to
	Type    string // The Response.Type (Like, Dislike, etc)
	Content string // Addional Value (for Emoji, etc)
	Exists  string // If TRUE, then create/update the response.  If FALSE, remove it.
}

func (txn *txnStepSetResponse) Bind(request *http.Request) error {

	// Parse values from Form
	values, err := formdata.Parse(request)

	if err != nil {
		return derp.Wrap(err, "build.txnStepSetResponse.Bind", "Error parsing form values")
	}

	// Populate data
	txn.URL = values.Get("url")
	txn.Type = values.Get("type")
	txn.Content = values.Get("content")
	txn.Exists = values.Get("exists")

	return nil
}
