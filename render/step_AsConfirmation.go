package render

import (
	"io"

	"github.com/benpate/html"
)

// StepAsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type StepAsConfirmation struct {
	Title   string
	Message string
	Submit  string
}

// Get displays a modal that asks users to continue or not.
func (step StepAsConfirmation) Get(renderer Renderer, buffer io.Writer) error {

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Content
	b.H1().InnerHTML(step.Title).Close()
	b.Div().Class("space-below").InnerHTML(step.Message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", renderer.URL()).Data("hx-swap", "none").InnerHTML(step.Submit).Close()
	b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	result := WrapModal(renderer.context().Response(), b.String())

	io.WriteString(buffer, result)
	return nil
}

func (step StepAsConfirmation) UseGlobalWrapper() bool {
	return false
}

// Post does nothing. (Other steps in the pipeline will make changes)
func (step StepAsConfirmation) Post(renderer Renderer) error {
	CloseModal(renderer.context(), "")
	return nil
}
