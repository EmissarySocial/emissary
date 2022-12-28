package render

import (
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tasks"
	"github.com/benpate/derp"
	"github.com/timewasted/go-accept-headers"
)

// StepWebSub represents an action-step that can render a Stream into HTML
type StepWebSub struct {
}

// Get renders the Stream HTML to the context
func (step StepWebSub) Get(renderer Renderer, buffer io.Writer) error {
	return nil
}

func (step StepWebSub) UseGlobalWrapper() bool {
	return true
}

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
	queue := factory.Queue()

	format, err := accept.Negotiate(renderer.context().Request().Header.Get("Accept"), model.MimeTypeJSONFeed, model.MimeTypeAtom, model.MimeTypeRSS, model.MimeTypeXML, model.MimeTypeXMLText)

	if err != nil {
		format = model.MimeTypeJSONFeed
	}

	go queue.Run(tasks.NewCreateWebSubFollower(
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
