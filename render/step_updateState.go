package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
)

type UpdateState struct {
	factory    Factory
	newStateID string
	form       form.Form
}

func NewUpdateState(factory Factory, command datatype.Map) UpdateState {

	return UpdateState{
		factory:    factory,
		newStateID: command.GetString("newStateID"),
		form:       form.MustParse(command.GetInterface("form")),
	}
}

// Get displays a form for users to fill out in the browser
func (action UpdateState) Get(renderer *Renderer) error {

	// Try to find the schema for the requested template
	templateService := action.factory.Template()
	schema, err := templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Get", "Invalid Schema")
	}

	// Try to render the form in HTML
	formLibrary := action.factory.FormLibrary()
	result, err := action.form.HTML(formLibrary, schema, renderer.stream)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Get", "Error generating form")
	}

	return renderer.ctx.HTML(http.StatusOK, result)
}

// Post updates the stream with configured data, and moves the stream to a new state
func (action UpdateState) Post(renderer *Renderer) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := renderer.ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := action.form.AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := renderer.stream.SetPath(p, body[p.String()]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error seting value", field)
		}
	}

	// Move stream to a new state
	renderer.stream.StateID = action.newStateID

	// Try to update the stream
	streamService := action.factory.Stream()
	if err := streamService.Save(&renderer.stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	renderer.ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+renderer.stream.Token+`"}}`)

	return renderer.ctx.NoContent(http.StatusOK)
}
