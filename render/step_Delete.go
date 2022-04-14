package render

import (
	"io"

	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepDelete represents an action-step that can delete a Stream from the Domain
type StepDelete struct {
	Title   string
	Message string
	Submit  string
}

// Get displays a customizable confirmation form for the delete
func (step StepDelete) Get(renderer Renderer, buffer io.Writer) error {

	b := html.New()

	b.H2().InnerHTML(step.Title).Close()
	b.Div().Class("space-below").InnerHTML(step.Message).Close()

	b.Button().Class("warning").
		Attr("hx-post", renderer.URL()).
		Attr("hx-swap", "none").
		InnerHTML(step.Submit).
		Close()

	b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()
	b.CloseAll()

	result := WrapModal(renderer.context().Response(), b.String())
	io.WriteString(buffer, result)

	return nil
}

func (step StepDelete) UseGlobalWrapper() bool {
	return true
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDelete) Post(renderer Renderer, buffer io.Writer) error {

	// Delete the object via the model service.
	if err := renderer.service().ObjectDelete(renderer.object(), "Deleted"); err != nil {
		return derp.Wrap(err, "render.StepDelete.Post", "Error deleting stream")
	}

	return nil
}
