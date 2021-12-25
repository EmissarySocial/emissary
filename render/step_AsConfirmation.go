package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/first"
	"github.com/benpate/html"
)

// StepAsConfirmation displays a confirmation dialog on GET, giving users an option to continue or not
type StepAsConfirmation struct {
	title   string
	message string
	submit  string
}

// NewStepAsConfirmation returns a fully initialized StepAsConfirmation object
func NewStepAsConfirmation(stepInfo datatype.Map) StepAsConfirmation {

	return StepAsConfirmation{
		title:   stepInfo.GetString("title"),
		message: stepInfo.GetString("message"),
		submit:  first.String(stepInfo.GetString("submit"), "Continue"),
	}
}

// Get displays a modal that asks users to continue or not.
func (step StepAsConfirmation) Get(buffer io.Writer, renderer Renderer) error {

	header := renderer.context().Response().Header()
	header.Set("HX-Retarget", "aside")
	header.Set("HX-Push", "false")

	b := html.New()

	// Modal Wrapper
	b.Div().ID("modal")
	b.Div().Class("modal-underlay").Script("on click send closeModal to #modal").Close()
	b.Div().Class("modal-content").EndBracket()

	b.H1().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()

	b.Div()
	b.Button().Class("primary").Data("hx-post", renderer.URL()).InnerHTML(step.submit).Close()
	b.Button().Script("on click trigger closeModal").InnerHTML("Cancel").Close()

	// Done
	b.CloseAll()

	// Copy the modal dialog into the response buffer
	if _, err := io.Copy(buffer, b.Reader()); err != nil {
		return derp.Wrap(err, "ghost.render.StepAsConfirmation.Get", "Error writing from builder to buffer")
	}

	return nil
}

// Post does nothing. (Other steps in the pipeline will make changes)
func (step StepAsConfirmation) Post(buffer io.Writer, renderer Renderer) error {
	closeModal(renderer.context(), "")
	return nil
}
