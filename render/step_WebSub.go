package render

import (
	"io"
	"net/http"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tasks"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/list"
	"github.com/timewasted/go-accept-headers"
)

// StepWebSub represents an action-step that can render a Stream into HTML
type StepWebSub struct {
}

// Get is not required by WebSub.  So let's redirect to the primary action.
func (step StepWebSub) Get(renderer Renderer, buffer io.Writer) error {
	newLocation := list.RemoveLast(renderer.URL(), list.DelimiterSlash)
	return renderer.context().Redirect(http.StatusSeeOther, newLocation)
}

func (step StepWebSub) UseGlobalWrapper() bool {
	return false
}

// Post accepts a WebSub request, verifies it, and potentially creates a new Follower record.
func (step StepWebSub) Post(renderer Renderer) error {

	var request struct {
		Mode         string `form:"hub.mode"`
		Topic        string `form:"hub.topic"`
		Callback     string `form:"hub.callback"`
		Secret       string `form:"hub.secret"`
		LeaseSeconds int    `form:"hub.lease_seconds"`
	}

	if err := renderer.context().Bind(&request); err != nil {
		return derp.Wrap(err, "render.StepWebSub.Post", "Error parsing form data")
	}

	// Try to validate and save the follower via the queue.
	factory := renderer.factory()

	format, err := accept.Negotiate(renderer.context().Request().Header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

	if err != nil {
		format = model.MimeTypeJSONFeed
	}

	// Run the task in the background queue.
	queue := factory.Queue()

	queue.Run(tasks.NewCreateWebSubFollower(
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

	// Set Status Code 202 (Accepted) to conform to WebSub spec
	// https://www.w3.org/TR/websub/#subscription-response-details
	renderer.context().Response().WriteHeader(202)

	return nil
}
