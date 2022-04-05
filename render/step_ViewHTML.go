package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	filename string

	BaseStep
}

// NewStepViewHTML generates a fully initialized StepViewHTML step.
func NewStepViewHTML(stepInfo datatype.Map) (StepViewHTML, error) {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return StepViewHTML{
		filename: filename,
	}, nil
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(_ Factory, renderer Renderer, buffer io.Writer) error {

	header := renderer.context().Response().Header()

	header.Set("Vary", "Cookie, HX-Request, User-Agent")
	header.Set("Cache-Control", "private")
	// object := renderer.object()
	// header.Set("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339))
	// header.Set("ETag", object.ETag())

	if err := renderer.executeTemplate(buffer, step.filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}
