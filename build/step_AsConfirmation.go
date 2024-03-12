package build

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
func (step StepAsConfirmation) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	// Modal Content
	b := html.New()
	b.H1().InnerText(step.Title).Close()
	b.Div().Class("margin-bottom").InnerText(step.Message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", builder.URL()).Data("hx-swap", "none").InnerText(step.Submit).Close()
	b.Button().Script("on click trigger closeModal").InnerText("Cancel").Close()

	// Done
	b.CloseAll()

	modalHTML := WrapModal(builder.response(), b.String())

	// nolint:errcheck
	io.WriteString(buffer, modalHTML)
	return Halt().AsFullPage()
}

// Post does nothing. (Other steps in the pipeline will make changes)
func (step StepAsConfirmation) Post(builder Builder, _ io.Writer) PipelineBehavior {
	return Continue().WithEvent("closeModal", "true")
}
