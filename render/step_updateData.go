package render

import (
	"net/http"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/path"
)

// UpdateData updates the specific data in a stream
type UpdateData struct {
	factory Factory
	form    form.Form
}

func NewUpdateData(factory Factory, command datatype.Map) UpdateData {

	return UpdateData{
		factory: factory,
		form:    form.MustParse(command.GetString("form")),
	}
}

// Get displays a form where users can update stream data
func (action UpdateData) Get(renderer *Renderer) error {

	templateService := action.factory.Template()
	formLibrary := action.factory.FormLibrary()

	// Try to find the schema for this Template
	schema, err := templateService.Schema(renderer.stream.TemplateID)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateData.Get", "Invalid Schema")
	}

	// Try to render the Form HTML
	result, err := action.form.HTML(formLibrary, schema, renderer.stream)

	if err != nil {
		return derp.Wrap(err, "ghost.render.UpdateData.Get", "Error generating form")
	}

	// Wrap result as a modal dialog
	return renderer.ctx.HTML(http.StatusOK, WrapModalForm(renderer, result))
}

// Post updates the stream with approved data from the request body.
func (action UpdateData) Post(renderer *Renderer) error {

	// Collect form POST information
	body := datatype.Map{}
	if err := renderer.ctx.Bind(&body); err != nil {
		return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateData.Post", "Error binding body")
	}

	// Put approved form data into the stream
	allPaths := action.form.AllPaths()
	for _, field := range allPaths {
		p := path.New(field.Path)
		if err := renderer.stream.SetPath(p, body[p.String()]); err != nil {
			return derp.New(derp.CodeBadRequestError, "ghost.render.UpdateData.Post", "Error seting value", field)
		}
	}

	// Try to update the stream
	streamService := action.factory.Stream()

	if err := streamService.Save(&renderer.stream, "Properties Updated"); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateData.Post", "Error updating state")
	}

	// Redirect the browser to the default page.
	renderer.ctx.Response().Header().Add("HX-Trigger", `{"closeModal":{"nextPage":"/`+renderer.stream.Token+`"}}`)
	return renderer.ctx.NoContent(http.StatusNoContent)
}
