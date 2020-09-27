package render

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/ghost/model"
)

type Form struct {
	templateService TemplateService
	library         form.Library
	stream          *model.Stream
	transition      string
}

func NewForm(templateService TemplateService, library form.Library, stream *model.Stream, transition string) Form {

	return Form{
		templateService: templateService,
		library:         library,
		stream:          stream,
		transition:      transition,
	}
}

func (w Form) Render() (string, error) {

	template, err := w.templateService.Load(w.stream.Template)

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Cannot load template"))
	}

	transition, err := template.Transition(w.stream.State, w.transition)

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Invalid Transition", w.transition))
	}

	// TODO: Validate that this transition is VALID
	// TODO: Validate that the USER IS PERMITTED to make this transition.

	if transition == nil {
		err = derp.New(404, "ghost.handler.GetTransition", "Unrecognized Transition", w.transition)
	}

	form, err := template.Form(w.stream.State, w.transition)

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Invalid Form", template))
	}

	// Generate HTML by merging the form with the element library, the data schema, and the data value
	html, err := form.HTML(w.library, *template.Schema, w.stream)

	if err != nil {
		return "", derp.Report(derp.Wrap(err, "ghost.handler.GetTransition", "Error generating form HTML", form))
	}

	return html, nil
}

func (w Form) Token() string {
	return w.stream.Token
}

func (w Form) StreamID() string {
	return w.stream.StreamID.String()
}

func (w Form) Label() string {
	return w.stream.Label
}
