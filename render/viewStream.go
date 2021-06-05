package render

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/benpate/steranko"
)

// ViewStream is an action that renders HTML for a stream.
type ViewStream struct {
	model.ActionConfig
}

// NewAction_ViewStream generates a fully initialized ViewStream action.
func NewAction_ViewStream(_ Factory, config model.ActionConfig) ViewStream {
	return ViewStream{
		ActionConfig: config,
	}
}

// Get renders the Stream HTML to the context
func (action ViewStream) Get(stream Renderer) (string, error) {

	var result bytes.Buffer

	t := action.template()

	if err := t.Execute(&result, stream); err != nil {
		return "", derp.Wrap(err, "ghost.render.ViewStream.Get", "Error executing template")
	}

	return result.String(), nil
}

// Post is not supported for this action.
func (action ViewStream) Post(ctx steranko.Context, stream *model.Stream) error {
	return derp.New(derp.CodeBadRequestError, "ghost.render.ViewStream.Get", "Unsupported Method")
}

// template retrieves the templpate paramer from the ActionConfig.
// IF this parameter is missing for some reason, it returns an empty template
func (action ViewStream) template() *template.Template {

	if t := action.GetInterface("template"); t != nil {

		if result, ok := t.(*template.Template); ok {
			return result
		}
	}

	return template.New("missing")
}
