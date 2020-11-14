package render

import (
	"bytes"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
	"github.com/davecgh/go-spew/spew"
)

// Form is a wrapper for a specific stream / template / transition that is Render()-ed into an HTML form
type Form struct {
	layoutService   LayoutService
	templateService TemplateService
	library         form.Library
	stream          model.Stream
	layout          string
	transition      string
}

// NewForm returns a fully populated Form wrapper object.
func NewForm(layoutService LayoutService, templateService TemplateService, library form.Library, stream model.Stream, layout string, transition string) Form {

	return Form{
		layoutService:   layoutService,
		templateService: templateService,
		library:         library,
		stream:          stream,
		layout:          layout,
		transition:      transition,
	}
}

// Render returns the HTML rendering of this Stream
func (w Form) Render() (string, error) {

	layout := w.layoutService.Layout()

	// TODO: Validate that this transition is VALID
	// TODO: Validate that the USER IS PERMITTED to make this transition.

	var result bytes.Buffer

	// Choose the correct view based on the wrapper provided.
	if err := layout.Funcs(sprig.FuncMap()).ExecuteTemplate(&result, w.layout, w); err != nil {
		return "", derp.Wrap(err, "ghost.render.Form.Render", "Error rendering view")
	}

	// Success!
	return result.String(), nil
}

// Token returns the Token for this Stream
func (w Form) Token() string {
	return w.stream.Token
}

// StreamID returns a string represenation of this StreamID
func (w Form) StreamID() string {
	return w.stream.StreamID.Hex()
}

// FormID returns the string representation of this FormID
func (w Form) FormID() string {
	return w.transition
}

// Label returns the publicly visible lable for this Form
func (w Form) Label() string {
	return w.stream.Label
}

// Form returns an HTML rendering of this form
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

	spew.Dump("FORM RESULT -----------", form, html)

	return template.HTML(html), nil
}

// SubTemplates returns a slice of templates that can be placed inside this Stream
func (w Form) SubTemplates() ([]model.Template) {
	return w.templateService.ListByContainer(w.stream.Template)
}
