package render

import (
	"bytes"
	"html/template"

	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
)

type Form struct {
	layout          *template.Template
	templateService TemplateService
	library         form.Library
	stream          *model.Stream
	wrapper         string
	transition      string
}

func NewForm(layout *template.Template, templateService TemplateService, library form.Library, stream *model.Stream, wrapper string, transition string) Form {

	return Form{
		layout:          layout,
		templateService: templateService,
		library:         library,
		stream:          stream,
		wrapper:         wrapper,
		transition:      transition,
	}
}

func (w Form) Render() (string, error) {

	layout, err := w.layout.Clone()

	if err != nil {
		return "", derp.Wrap(err, "ghost.render.Form.Render", "Error cloning template")
	}

	// TODO: Validate that this transition is VALID
	// TODO: Validate that the USER IS PERMITTED to make this transition.

	var result bytes.Buffer

	// Choose the correct view based on the wrapper provided.
	if err := layout.ExecuteTemplate(&result, w.wrapper, w); err != nil {
		return "", derp.Wrap(err, "ghost.render.Form.Render", "Error rendering view")
	}

	// Success!
	return result.String(), nil
}

func (w Form) Token() string {
	return w.stream.Token
}

func (w Form) StreamID() string {
	return w.stream.StreamID.String()
}

func (w Form) FormID() string {
	return w.transition
}

func (w Form) Label() string {
	return w.stream.Label
}

func (w Form) Form() (template.HTML, error) {

	var result template.HTML

	t, err := w.templateService.Load(w.stream.Template)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Cannot load template"))
	}

	// TODO: Validate that this transition is VALID
	// TODO: Validate that the USER IS PERMITTED to make this transition.

	form, err := t.Form(w.stream.State, w.transition)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Invalid Form", t))
	}

	// Generate HTML by merging the form with the element library, the data schema, and the data value
	html, err := form.HTML(w.library, *t.Schema, w.stream)

	if err != nil {
		return result, derp.Report(derp.Wrap(err, "ghost.handler.GetForm", "Error generating form HTML", form))
	}

	return template.HTML(html), nil
}
