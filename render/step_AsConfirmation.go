package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/first"
	"github.com/benpate/html"
)

// StepAsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type StepAsConfirmation struct {
	title   string
	message string
	submit  string

	BaseStep
}

// NewStepAsConfirmation returns a fully initialized StepAsConfirmation object
func NewStepAsConfirmation(stepInfo datatype.Map) (StepAsConfirmation, error) {

	return StepAsConfirmation{
		title:   stepInfo.GetString("title"),
		message: stepInfo.GetString("message"),
		submit:  first.String(stepInfo.GetString("submit"), "Continue"),
	}, nil
}

// Get displays a modal that asks users to continue or not.
func (step StepAsConfirmation) Get(_ Factory, renderer Renderer, buffer io.Writer) error {

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Content
	b.H1().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", renderer.URL()).Data("hx-swap", "none").InnerHTML(step.submit).Close()
	b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	result := WrapModal(renderer.context().Response(), b.String())

	io.WriteString(buffer, result)
	return nil
}

// Post does nothing. (Other steps in the pipeline will make changes)
func (step StepAsConfirmation) Post(_ Factory, renderer Renderer, buffer io.Writer) error {
	CloseModal(renderer.context(), "")
	return nil
}
