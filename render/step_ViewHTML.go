package render

import (
	"io"

	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	Filename string
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(renderer Renderer, buffer io.Writer) error {

	header := renderer.context().Response().Header()

	header.Set("Vary", "Cookie, HX-Request, User-Agent")
	header.Set("Cache-Control", "private")

	var filename string

	if step.Filename != "" {
		filename = step.Filename
	} else {
		filename = renderer.ActionID()
	}

	// object := renderer.object()
	// header.Set("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339))
	// header.Set("ETag", object.ETag())

	if err := renderer.executeTemplate(buffer, filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

func (step StepViewHTML) Post(renderer Renderer, buffer io.Writer) error {
	return nil
}
