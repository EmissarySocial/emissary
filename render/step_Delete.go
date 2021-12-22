package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/html"
)

// StepDelete represents an action-step that can delete a Stream from the Domain
type StepDelete struct {
	modelService ModelService
	title        string
	message      string
}

// NewStepDelete returns a fully populated StepDelete object
func NewStepDelete(modelService ModelService, stepInfo datatype.Map) StepDelete {
	return StepDelete{
		modelService: modelService,
		title:        stepInfo.GetString("title"),
		message:      stepInfo.GetString("message"),
	}
}

// Get displays a customizable confirmation form for the delete
func (step StepDelete) Get(buffer io.Writer, renderer Renderer) error {

	if step.title == "" {
		step.title = "Confirm Delete"
	}

	if step.message == "" {
		step.message = "Are you sure you want to delete this item?  There is NO UNDO."
	}

	b := html.New()

	b.Div().ID("modal").Data("HX-Push-Url", "false")
	b.Div().Class("modal-underlay").Close()
	b.Div().Class("modal-content")
	b.H2().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()

	b.Button().Class("warning").
		Attr("hx-post", renderer.URL()).
		Attr("hx-swap", "none").
		InnerHTML("Delete").Close()

	b.Button().Script("install ModalCancelButton()").InnerHTML("Cancel").Close()
	b.CloseAll()

	buffer.Write([]byte(b.String()))

	return nil
}

// Post removes the object from the database (likely using a soft-delete, though)
func (step StepDelete) Post(buffer io.Writer, renderer Renderer) error {

	// Delete the object via the model service.
	if err := step.modelService.ObjectDelete(renderer.object(), "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.StepDelete.Post", "Error deleting stream")
	}

	return nil
}
