package render

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/service"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/timewasted/go-accept-headers"
)

// StepWebSub represents an action-step that can render a Stream into HTML
type StepWebSub struct {
}

// Get is not required by WebSub.  So let's redirect to the primary action.
func (step StepWebSub) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	// TODO: MEDIUM: This may not jive with the new PipelineBehavior model.  Check accordingly.
	newLocation := list.RemoveLast(renderer.URL(), list.DelimiterSlash)
	if err := renderer.context().Redirect(http.StatusSeeOther, newLocation); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepWebSub.Get", "Error writing redirection", newLocation))
	}
	return nil
}

// Post accepts a WebSub request, verifies it, and potentially creates a new Follower record.
func (step StepWebSub) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	var request struct {
		Mode         string `form:"hub.mode"`
		Topic        string `form:"hub.topic"`
		Callback     string `form:"hub.callback"`
		Secret       string `form:"hub.secret"`
		LeaseSeconds int    `form:"hub.lease_seconds"`
	}

	if err := renderer.context().Bind(&request); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepWebSub.Post", "Error parsing form data"))
	}

	// Try to validate and save the follower via the queue.
	factory := renderer.factory()

	format, err := accept.Negotiate(renderer.context().Request().Header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

	if err != nil {
		format = model.MimeTypeJSONFeed
	}

	// Run the task in the background queue.
	factory.Queue().Run(service.NewTaskCreateWebSubFollower(
		factory.Follower(),
		factory.Locator(),
		renderer.objectType(),
		renderer.objectID(),
		format,
		request.Mode,
		request.Topic,
		request.Callback,
		request.Secret,
		request.LeaseSeconds,
	))

	// TODO: MEDIUM: This may not jive with the new PipelineBehavior model.  Check accordingly.
	// Set Status Code 202 (Accepted) to conform to WebSub spec
	// https://www.w3.org/TR/websub/#subscription-response-details
	renderer.context().Response().WriteHeader(202)

	return nil
}
