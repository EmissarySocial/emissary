package render

import (
	"io"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepDelete represents an action-step that can delete a Stream from the Domain
type StepDelete struct {
	Title   *template.Template
	Message *template.Template
	Submit  string
}

// Get displays a customizable confirmation form for the delete
func (step StepDelete) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	b := html.New()

	b.H1().InnerText(executeTemplate(step.Title, renderer)).Close()
	b.Div().Class("margin-bottom").InnerText(executeTemplate(step.Message, renderer)).Close()

	b.Button().Class("warning").
		Attr("hx-post", renderer.URL()).
		Attr("hx-swap", "none").
		Attr("hx-push-url", "false").
		InnerText(step.Submit).
		Close()

	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()
	b.CloseAll()

	modalHTML := WrapModal(renderer.response(), b.String())

	// nolint:errcheck
	io.WriteString(buffer, modalHTML)

	return Halt().AsFullPage()
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDelete) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	// Delete the object via the model service.
	if err := renderer.service().ObjectDelete(renderer.object(), "Deleted"); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepDelete.Post", "Error deleting stream"))
	}

	return Continue().WithEvent("closeModal", "true")
}
