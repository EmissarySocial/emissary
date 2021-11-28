package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/service"
	"github.com/benpate/path"
)

// StepStreamData represents an action-step that can update the data.DataMap custom data stored in a Stream
type StepStreamData struct {
	templateService *service.Template
	streamService   *service.Stream
	formLibrary     form.Library
	form            form.Form
}

func NewStepStreamData(templateService *service.Template, streamService *service.Stream, formLibrary form.Library, command datatype.Map) StepStreamData {

	return StepStreamData{
		templateService: templateService,
		streamService:   streamService,
		formLibrary:     formLibrary,
		form:            form.MustParse(command.GetString("form")),
	}
}

// Get displays a form where users can update stream data
func (step StepStreamData) Get(buffer io.Writer, renderer *Renderer) error {

	// Try to find the schema for this Template
	schema, err := step.templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamData.Get", "Invalid Schema")
	}

	// Try to render the Form HTML
	result, err := step.form.HTML(step.formLibrary, schema, renderer.stream)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamData.Get", "Error generating form")
	}

	// Wrap result as a modal dialog
	buffer.Write([]byte(WrapModalForm(renderer, result)))
	return nil
}

// Post updates the stream with approved data from the request body.
func (step StepStreamData) Post(buffer io.Writer, renderer *Renderer) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := renderer.ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamData.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := step.form.AllPaths()
	for _, field := range allPaths {
		if err := path.Set(renderer.stream, field.Path, body[field.Path]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepStreamData.Post", "Error seting value", field)
		}
	}

	// Try to update the stream

	if err := step.streamService.Save(renderer.stream, "Properties Updated"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamData.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	renderer.ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+renderer.stream.Token+`"}}`)
	return nil
}
