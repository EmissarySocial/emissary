package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// ViewStream renders HTML for a stream.
type ViewStream struct {
	filename string
}

// NewAction_ViewStream generates a fully initialized ViewStream step.
func NewViewStream(stepInfo datatype.Map) ViewStream {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return ViewStream{
		filename: filename,
	}
}

// Get renders the Stream HTML to the context
func (step ViewStream) Get(buffer io.Writer, renderer *Renderer) error {

	template, ok := renderer.template.HTMLTemplate(step.filename)

	if !ok {
		return derp.New(derp.CodeBadRequestError, "ghost.renderer.ViewStream.Get", "Cannot find template", step.filename)
	}

	if err := template.Execute(buffer, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.ViewStream.Get", "Error executing template")
	}

	return nil
}

// Post is not supported for this step.
func (step ViewStream) Post(buffer io.Writer, renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.ViewStream.Get", "Unsupported Method")
}
