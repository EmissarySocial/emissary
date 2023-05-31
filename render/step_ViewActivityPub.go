package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
)

// StepViewActivityPub represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepViewActivityPub struct {
	File string
}

func (step StepViewActivityPub) Get(renderer Renderer, buffer io.Writer) error {

	// Try to load the uri from the Internet
	client := renderer.factory().ActivityStreams()
	uri := renderer.context().QueryParam("uri")

	document, err := streams.NewDocument(uri, streams.WithClient(client)).Load()

	if err != nil {
		return derp.Wrap(err, "render.StepViewActivityPub.Get", "Error loading document from the internet")
	}

	if err := renderer.executeTemplate(buffer, step.File, document); err != nil {
		return derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

func (step StepViewActivityPub) UseGlobalWrapper() bool {
	return true
}

// Post updates the stream with approved data from the request body.
func (step StepViewActivityPub) Post(renderer Renderer, _ io.Writer) error {
	return nil
}
