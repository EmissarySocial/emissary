package build

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/davecgh/go-spew/spew"
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

	spew.Dump(location, transaction)

	// Retrieve the currently authenticated user
	user, err := builder.getUser()

	if err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error getting user"))
	}

	spew.Dump("A")

	// Set the value in the database
	responseService := builder.factory().Response()
	spew.Dump("B")

	// Create/Update the response
	if transaction.Exists {
		spew.Dump("C")

		if err := responseService.SetResponse(user, transaction.URL, transaction.Type, transaction.Content); err != nil {
			spew.Dump("D")
			return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
		}
		spew.Dump("E")

		return Continue()
	}
	spew.Dump("F")

	// Fall through means DELETE the Response
	if err := responseService.UnsetResponse(user, transaction.URL, transaction.Type); err != nil {
		spew.Dump("G")
		return Halt().WithError(derp.Wrap(err, location, "Error setting response"))
	}
	spew.Dump("H")

	// Carry on, carry onnnnn...
	return Continue()
}

type txnStepSetResponse struct {
	URL     string // The URL of the object being responded to
	Type    string // The Response.Type (Like, Dislike, etc)
	Content string // Addional Value (for Emoji, etc)
	Exists  bool   // If TRUE, then create/update the response.  If FALSE, remove it.
}

func (txn *txnStepSetResponse) Bind(request *http.Request) error {

	const location = "build.txnStepSetResponse.Bind"

	// Parse values from Form
	values, err := formdata.Parse(request)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing form values")
	}

	spew.Dump(location, values)

	// Populate data
	if url := values.Get("url"); url == "" {
		return derp.ValidationError("The 'url' field cannot be empty.")
	} else {
		txn.URL = url
	}

	if responseType := values.Get("type"); responseType == "" {
		return derp.ValidationError("The 'type' field cannot be empty.")
	} else {
		txn.Type = responseType
	}

	txn.Content = values.Get("content")
	txn.Exists = convert.Bool(values.Get("exists"))

	return nil
}
