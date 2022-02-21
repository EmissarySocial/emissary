package render

import (
	"io"
	"time"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// StepViewHTML represents an action-step that can render a Stream into HTML
type StepViewHTML struct {
	filename string
}

// NewStepViewHTML generates a fully initialized StepViewHTML step.
func NewStepViewHTML(stepInfo datatype.Map) StepViewHTML {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return StepViewHTML{
		filename: filename,
	}
}

// Get renders the Stream HTML to the context
func (step StepViewHTML) Get(buffer io.Writer, renderer Renderer) error {

	header := renderer.context().Response().Header()
	object := renderer.object()

	header.Set("Vary", "Cookie, HX-Request, User-Agent")
	header.Set("Cache-Control", "private")
	header.Set("Last-Modified", time.UnixMilli(object.Updated()).Format(time.RFC3339))
	header.Set("ETag", object.ETag())

	if err := renderer.executeTemplate(buffer, step.filename, renderer); err != nil {
		return derp.Wrap(err, "whisper.render.StepViewHTML.Get", "Error executing template")
	}

	return nil
}

// Post is not supported for this step.
func (step StepViewHTML) Post(buffer io.Writer, renderer Renderer) error {
	return nil
}
