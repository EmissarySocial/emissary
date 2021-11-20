package render

import (
	"bytes"
	"html/template"
	"math/rand"
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content/transaction"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// UpdateDraft manages the content.Content in a stream.
type UpdateDraft struct {
	draftService *service.StreamDraft
	template     *template.Template
}

func NewUpdateDraft(draftService *service.StreamDraft, command datatype.Map) UpdateDraft {
	return UpdateDraft{
		draftService: draftService,
		template:     mustTemplate(command.GetInterface("template")),
	}
}

func (step UpdateDraft) Get(renderer *Renderer) error {

	var result bytes.Buffer

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.LoadByID(renderer.stream.StreamID, &renderer.stream); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	if err := step.template.Execute(&result, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.UpdateDraft.Get", "Error executing template")
	}

	return renderer.ctx.HTML(http.StatusOK, result.String())
}

func (step UpdateDraft) Post(renderer *Renderer) error {

	var draft model.Stream

	// Try to load the stream draft from the database
	if err := step.draftService.LoadByID(renderer.stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.UpdateDraft.Post", "Error loading Draft")
	}

	// Try to parse the body content into a transaction
	body := make(map[string]interface{})

	if err := renderer.ctx.Bind(&body); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error binding data"))
	}

	txn, err := transaction.Parse(body)

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error parsing transaction", body))
	}

	// Try to execute the transaction
	if err := txn.Execute(&(draft.Content)); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error executing transaction", txn))
	}

	// Try to save the draft

	if err := step.draftService.Save(&draft, "edit content: "+txn.Description()); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.PostStreamDraft", "Error saving stream"))
	}

	// Return response to caller
	return renderer.ctx.String(http.StatusOK, convert.String(rand.Int63()))

	// ctx.Response().Header().Add("HX-Redirect", "/"+stream.Token)
	// return ctx.NoContent(200)

}
