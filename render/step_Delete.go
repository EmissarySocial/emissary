package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/html"
)

// StepDelete represents an action-step that can delete a Stream from the Domain
type StepDelete struct {
	title   string
	message string
	submit  string

	BaseStep
}

// NewStepDelete returns a fully populated StepDelete object
func NewStepDelete(stepInfo datatype.Map) (StepDelete, error) {
	return StepDelete{
		title:   first.String(stepInfo.GetString("title"), "Confirm Delete"),
		message: first.String(stepInfo.GetString("message"), "Are you sure you want to delete this item?  There is NO UNDO."),
		submit:  first.String(stepInfo.GetString("submit"), "Delete"),
	}, nil
}

// Get displays a customizable confirmation form for the delete
func (step StepDelete) Get(_ Factory, renderer Renderer, buffer io.Writer) error {

	b := html.New()

	b.H2().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()

	b.Button().Class("warning").
		Attr("hx-post", renderer.URL()).
		Attr("hx-swap", "none").
		InnerHTML(step.submit).
		Close()

	b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()
	b.CloseAll()

	result := WrapModal(renderer.context().Response(), b.String())
	io.WriteString(buffer, result)

	return nil
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDelete) Post(_ Factory, renderer Renderer, buffer io.Writer) error {

	// Delete the object via the model service.
	if err := renderer.service().ObjectDelete(renderer.object(), "Deleted"); err != nil {
		return derp.Wrap(err, "whisper.render.StepDelete.Post", "Error deleting stream")
	}

	return nil
}
