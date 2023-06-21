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
func (step StepAsConfirmation) Get(renderer Renderer, buffer io.Writer) ExitCondition {

	// Modal Content
	b := html.New()
	b.H1().InnerText(step.Title).Close()
	b.Div().Class("space-below").InnerText(step.Message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", renderer.URL()).Data("hx-swap", "none").InnerText(step.Submit).Close()
	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()

	// Done
	b.CloseAll()

	result := WrapModal(renderer.context().Response(), b.String())

	// nolint:errcheck
	io.WriteString(buffer, result)
	return ExitFullPage()
}

// Post does nothing. (Other steps in the pipeline will make changes)
func (step StepAsConfirmation) Post(renderer Renderer, _ io.Writer) ExitCondition {
	CloseModal(renderer.context(), "")
	return nil
}
