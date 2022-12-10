package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	File string
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(renderer Renderer, buffer io.Writer) error {

	header := renderer.context().Response().Header()

	header.Set("Vary", "Cookie, HX-Request")
	header.Set("Cache-Control", "private")

	var filename string

	if step.File != "" {
		filename = step.File
	} else {
		filename = renderer.ActionID()
	}

	// TODO: MEDIUM: Re-implement caching.  Will need to automatically compute the "Vary" header.
	// object := renderer.object()
	// header.Set("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339))
	// header.Set("ETag", object.ETag())

	if err := renderer.executeTemplate(buffer, filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

func (step StepViewHTML) UseGlobalWrapper() bool {
	return true
}

func (step StepViewHTML) Post(renderer Renderer) error {
	return nil
}
