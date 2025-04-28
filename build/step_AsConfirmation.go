package build

import (
	"io"

	"github.com/benpate/html"
)

// StepAsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type StepAsConfirmation struct {
	Icon    string
	Title   string
	Message string
	Submit  string
}

// Get displays a modal that asks users to continue or not.
func (step StepAsConfirmation) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Icon != "" {
		iconService := builder.factory().Icons()
		step.Title = iconService.Get(step.Icon) + " " + step.Title
	}

	// Modal Content
	b := html.New()
	b.H1().InnerHTML(step.Title).Close()
	b.Div().Class("margin-bottom").InnerHTML(step.Message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", builder.URL()).Data("hx-swap", "none").Data("hx-push-url", "false").InnerText(step.Submit).Close()
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
