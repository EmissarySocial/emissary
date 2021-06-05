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

type UpdateState struct {
	factory Factory
	model.ActionConfig
}

func NewAction_UpdateState(factory Factory, config model.ActionConfig) UpdateState {

	return UpdateState{
		factory:      factory,
		ActionConfig: config,
	}
}

// Get displays a form for users to fill out in the browser
func (action UpdateState) Get(renderer Renderer) (string, error) {

	// Try to find the schema for the requested template
	templateService := action.factory.Template()
	schema, err := templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.UpdateState.Get", "Invalid Schema")
	}

	// Try to render the form in HTML
	formLibrary := action.factory.FormLibrary()
	result, err := action.form().HTML(formLibrary, schema, renderer.stream)

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.UpdateState.Get", "Error generating form")
	}

	return result, nil
}

// Post updates the stream with configured data, and moves the stream to a new state
func (action UpdateState) Post(ctx steranko.Context, stream *model.Stream) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := action.form().AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := stream.SetPath(p, body); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error seting value", field)
		}
	}

	// Move stream to a new state
	stream.StateID = action.newStateID()

	// Try to update the stream
	streamService := action.factory.Stream()
	if err := streamService.Save(stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+stream.Token+`"}}`)

	return ctx.NoContent(http.StatusOK)
}

// form retrieves the form parameter from the ActionConfig
func (action UpdateState) form() form.Form {
	result, err := form.Parse(action.GetInterface("form"))

	if err != nil {
		derp.Report(err)
	}

	return result
}

// newStateID retrieves the newStateID parameter from the ActionConfig
func (action UpdateState) newStateID() string {
	return action.GetString("newStateId")
}
