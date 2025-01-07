package build

import (
	"io"
	"net/http"
	"strconv"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/turbine/queue"
	"github.com/timewasted/go-accept-headers"
)

// StepWebSub is a Step that can build a Stream into HTML
type StepWebSub struct {
}

// Get is not required by WebSub.  So let's redirect to the primary action.
func (step StepWebSub) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	// TODO: MEDIUM: This may not jive with the new PipelineBehavior model.  Check accordingly.
	newLocation := list.RemoveLast(builder.URL(), list.DelimiterSlash)
	if err := redirect(builder.response(), http.StatusSeeOther, newLocation); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepWebSub.Get", "Error writing redirection", newLocation))
	}
	return nil
}

// Post accepts a WebSub request, verifies it, and potentially creates a new Follower record.
func (step StepWebSub) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepWebSub.Post"

	// This transaction will capture the form POST input
	var transaction txnStepWebSub

	// Try to collect form data into the transaction struct
	if err := transaction.Bind(builder.request()); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error parsing form data"))
	}

	// Negotiate the content type (format) requested by the WebSub follower
	format, err := accept.Negotiate(builder.request().Header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

	if err != nil {
		format = model.MimeTypeJSONFeed
	}

	// Create a new background task to handle the WebSub follower
	task := queue.NewTask("CreateWebSubFollower", mapof.Any{
		"objectType":   builder.objectType(),
		"objectId":     builder.objectID(),
		"format":       format,
		"mode":         transaction.Mode,
		"topic":        transaction.Topic,
		"callback":     transaction.Callback,
		"secret":       transaction.Secret,
		"leaseSeconds": transaction.LeaseSeconds,
	})

	// Push the new task onto the background queue.
	if err := builder.factory().Queue().Publish(task); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Error pushing task to queue"))
	}

	// TODO: MEDIUM: This may not jive with the new PipelineBehavior model.  Check accordingly.
	// Set Status Code 202 (Accepted) to conform to WebSub spec
	// https://www.w3.org/TR/websub/#subscription-response-details
	builder.response().WriteHeader(202)

	return nil
}

type txnStepWebSub struct {
	Mode         string
	Topic        string
	Callback     string
	Secret       string
	LeaseSeconds int
}

func (txn *txnStepWebSub) Bind(request *http.Request) error {

	const location = "build.txnStepWebSub.Bind"

	// Parse Form Values
	values, err := formdata.Parse(request)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing form values")
	}

	// Apply values to this object
	txn.Mode = values.Get("hub.mode")
	txn.Topic = values.Get("hub.topic")
	txn.Callback = values.Get("hub.callback")
	txn.Secret = values.Get("hub.secret")
	txn.LeaseSeconds, err = strconv.Atoi(values.Get("hub.lease_seconds"))

	if err != nil {
		return derp.Wrap(err, location, "Error parsing integer", values.Get("hub.lease_seconds"))
	}

	return nil
}
