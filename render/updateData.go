package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/benpate/path"
	"github.com/benpate/steranko"
)

// UpdateData updates the specific data in a stream
type UpdateData struct {
	factory Factory
	model.ActionConfig
}

func NewAction_UpdateData(factory Factory, config model.ActionConfig) UpdateData {

	return UpdateData{
		factory:      factory,
		ActionConfig: config,
	}
}

// Get displays a form where users can update stream data
func (action UpdateData) Get(renderer Renderer) (string, error) {

	templateService := action.factory.Template()
	formLibrary := action.factory.FormLibrary()

	// Try to find the schema for this Template
	schema, err := templateService.Schema(renderer.TemplateID())

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.UpdateData.Get", "Invalid Schema")
	}

	// Try to render the Form HTML
	result, err := action.form().HTML(formLibrary, schema, renderer.stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.UpdateData.Get", "Error generating form")
	}

	return result, nil
}

// Post updates the stream with approved data from the request body.
func (action UpdateData) Post(ctx steranko.Context, stream *model.Stream) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateData.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := action.form().AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := stream.SetPath(p, body); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateData.Post", "Error seting value", field)
		}
	}

	// Try to update the stream
	streamService := action.factory.Stream()

	if err := streamService.Save(stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateData.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+stream.Token+`"}}`)
	return ctx.NoContent(http.StatusOK)
}

// form extracts the embedded form from the action.  It reports errors, but
// does not return them, because these should only happen during development,
// not on live sites.
func (action UpdateData) form() form.Form {
	result, err := form.Parse(action.GetInterface("form"))

	if err != nil {
		derp.Report(err)
	}

	return result
}
