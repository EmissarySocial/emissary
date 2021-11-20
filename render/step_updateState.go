package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
)

type UpdateState struct {
	templateService *service.Template
	streamService   *service.Stream
	formLibrary     form.Library
	newStateID      string
	form            form.Form
}

func NewUpdateState(templateService *service.Template, streamService *service.Stream, formLibrary form.Library, stepInfo datatype.Map) UpdateState {

	return UpdateState{
		templateService: templateService,
		streamService:   streamService,
		newStateID:      stepInfo.GetString("newStateID"),
		form:            form.MustParse(stepInfo.GetInterface("form")),
	}
}

// Get displays a form for users to fill out in the browser
func (step UpdateState) Get(renderer *Renderer) error {

	// Try to find the schema for the requested template
	schema, err := step.templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Get", "Invalid Schema")
	}

	// Try to render the form in HTML
	result, err := step.form.HTML(step.formLibrary, schema, renderer.stream)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Get", "Error generating form")
	}

	return renderer.ctx.HTML(http.StatusOK, result)
}

// Post updates the stream with configured data, and moves the stream to a new state
func (step UpdateState) Post(renderer *Renderer) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := renderer.ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := step.form.AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := renderer.stream.SetPath(p, body[p.String()]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateState.Post", "Error seting value", field)
		}
	}

	// Move stream to a new state
	renderer.stream.StateID = step.newStateID

	// Try to update the stream
	if err := step.streamService.Save(&renderer.stream, "Moved to new State"); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateState.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	renderer.ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+renderer.stream.Token+`"}}`)

	return renderer.ctx.NoContent(http.StatusOK)
}
