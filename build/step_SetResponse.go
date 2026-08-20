package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
)

// StepSetResponse is a Step that records a User's Like, Dislike, or Announce of a remote object
type StepSetResponse struct{}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepSetResponse) Get(builder Builder, buffer io.Writer) PipelineBehavior {
	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepSetResponse) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepSetResponse.Post"

	// Receive the transaction data
	transaction := txnStepSetResponse{}

	if err := transaction.Bind(builder.request()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Binding transaction"))
	}

	// Retrieve the currently authenticated user
	user, err := builder.getUser()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Getting user"))
	}

	// Set the value in the database
	responseService := builder.factory().Response()

	// Create/Update the response
	if transaction.Exists {

		if err := responseService.SetResponse(builder.session(), user, transaction.URL, transaction.Type, transaction.Content); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Setting response"))
		}

		return Continue()
	}

	// Fall through means DELETE the Response
	if err := responseService.UnsetResponse(builder.session(), user, transaction.URL, transaction.Type); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Setting response"))
	}

	// Carry on, carry onnnnn...
	return Continue()
}

// txnStepSetResponse is the form posted to a "set-response" step
type txnStepSetResponse struct {
	URL     string // The URL of the object being responded to
	Type    string // The Response.Type (Like, Dislike, etc)
	Content string // Addional Value (for Emoji, etc)
	Exists  bool   // If TRUE, then create/update the response.  If FALSE, remove it.
}

// Bind reads this transaction from the request form values
func (txn *txnStepSetResponse) Bind(request *http.Request) error {

	const location = "build.txnStepSetResponse.Bind"

	// Parse values from Form
	values, err := formdata.Parse(request)

	if err != nil {
		return derp.Wrap(err, location, "Parsing form values")
	}

	// Populate data
	if url := values.Get("url"); url == "" {
		return derp.Validation("The 'url' field cannot be empty.")
	} else {
		txn.URL = url
	}

	if responseType := values.Get("type"); responseType == "" {
		return derp.Validation("The 'type' field cannot be empty.")
	} else {
		txn.Type = responseType
	}

	txn.Content = values.Get("content")
	txn.Exists = convert.Bool(values.Get("exists"))

	return nil
}
