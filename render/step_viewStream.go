package render

import (
	"bytes"
	"net/http"

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
func (step ViewStream) Get(renderer *Renderer) error {

	var result bytes.Buffer

	template, ok := renderer.template.HTMLTemplate(step.filename)

	if !ok {
		return derp.New(derp.CodeBadRequestError, "ghost.renderer.ViewStream.Get", "Cannot find template", step.filename)
	}

	if err := template.Execute(&result, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.ViewStream.Get", "Error executing template")
	}

	return renderer.ctx.HTML(http.StatusOK, result.String())
}

// Post is not supported for this step.
func (step ViewStream) Post(renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.ViewStream.Get", "Unsupported Method")
}
