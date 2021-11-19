package render

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
)

// ViewStream renders HTML for a stream.
type ViewStream struct {
	template *template.Template
}

// NewAction_ViewStream generates a fully initialized ViewStream step.
func NewViewStream(_ Factory, command datatype.Map) ViewStream {
	result := ViewStream{}
	t := command.GetInterface("template")

	if t, ok := t.(*template.Template); ok {
		result.template = t
	}

	return result
}

// Get renders the Stream HTML to the context
func (step ViewStream) Get(renderer *Renderer) error {

	var result bytes.Buffer

	t := step.template

	if err := t.Execute(&result, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.ViewStream.Get", "Error executing template")
	}

	return renderer.ctx.HTML(http.StatusOK, result.String())
}

// Post is not supported for this step.
func (step ViewStream) Post(renderer *Renderer) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.ViewStream.Get", "Unsupported Method")
}
