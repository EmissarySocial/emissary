package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/service"
	"github.com/benpate/html"
)

// StepStreamDelete represents an action-step that can delete a Stream from the Domain
type StepStreamDelete struct {
	modelService ModelService
	title        string
	message      string
}

func NewStepStreamDelete(modelService ModelService, draftService *service.StreamDraft, stepInfo datatype.Map) StepStreamDelete {
	return StepStreamDelete{
		modelService: modelService,
		title:        stepInfo.GetString("title"),
		message:      stepInfo.GetString("message"),
	}
}

func (step StepStreamDelete) Get(buffer io.Writer, renderer Renderer) error {

	if step.title == "" {
		step.title = "Confirm Delete"
	}

	if step.message == "" {
		step.message = "Are you sure you want to delete this item?  There is NO UNDO."
	}

	b := html.New()

	b.Div().ID("modal").Data("HX-Push-Url", "false")
	b.Div().Class("modal-backdrop").Close()
	b.Div().Class("modal-content")
	b.H2().InnerHTML(step.title).Close()
	b.Div().Class("space-below").InnerHTML(step.message).Close()

	b.Button().Class("warning").
		Attr("hx-post", "/"+renderer.object().ID()+"/delete").
		Attr("hx-swap", "none").
		Script("install SubmitButton()").
		InnerHTML("Delete").Close()

	b.Button().Script("install ModalCancelButton()").InnerHTML("Cancel").Close()
	b.CloseAll()

	buffer.Write([]byte(b.String()))

	return nil
}

func (step StepStreamDelete) Post(buffer io.Writer, renderer Renderer) error {

	if err := step.modelService.ObjectDelete(renderer.object(), "Deleted"); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDelete.Post", "Error deleting stream")
	}

	return nil
}
