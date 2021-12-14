package render

import (
	"io"
	"math/rand"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/content/transaction"
	"github.com/benpate/ghost/model"
	"github.com/benpate/ghost/service"
)

// StepStreamDraftEdit represents an action-step that can edit/update content.Content in a streamDraft.
type StepStreamDraftEdit struct {
	draftService *service.StreamDraft
	filename     string
}

func NewStepStreamDraftEdit(draftService *service.StreamDraft, stepInfo datatype.Map) StepStreamDraftEdit {

	filename := stepInfo.GetString("file")

	if filename == "" {
		filename = stepInfo.GetString("actionId")
	}

	return StepStreamDraftEdit{
		draftService: draftService,
		filename:     filename,
	}
}

func (step StepStreamDraftEdit) Get(buffer io.Writer, renderer Renderer) error {

	streamRenderer := renderer.(Stream)

	// Try to load the draft from the database, overwriting the stream already in the renderer
	if err := step.draftService.LoadByID(streamRenderer.stream.StreamID, &streamRenderer.stream); err != nil {
		return derp.Wrap(err, "ghost.renderer.StepStreamDraftEdit.Get", "Error loading Draft")
	}

	if err := streamRenderer.executeTemplate(buffer, step.filename, streamRenderer); err != nil {
		return derp.Wrap(err, "ghost.render.StepStreamDraftEdit.Get", "Error executing template")
	}

	return nil
}

func (step StepStreamDraftEdit) Post(buffer io.Writer, renderer Renderer) error {

	var draft model.Stream
	streamRenderer := renderer.(Stream)

	// Try to load the stream draft from the database
	if err := step.draftService.LoadByID(streamRenderer.stream.StreamID, &draft); err != nil {
		return derp.Wrap(err, "ghost.renderer.StepStreamDraftEdit.Post", "Error loading Draft")
	}

	// Try to parse the body content into a transaction
	body := make(map[string]interface{})

	if err := streamRenderer.ctx.Bind(&body); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftEdit.Post", "Error binding data"))
	}

	txn, err := transaction.Parse(body)

	if err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftEdit.Post", "Error parsing transaction", body))
	}

	// Try to execute the transaction
	if err := txn.Execute(&(draft.Content)); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftEdit.Post", "Error executing transaction", txn))
	}

	// Try to save the draft

	if err := step.draftService.Save(&draft, "edit content: "+txn.Description()); err != nil {
		return derp.Report(derp.Wrap(err, "ghost.handler.StepStreamDraftEdit.Post", "Error saving stream"))
	}

	// Return response to caller
	buffer.Write([]byte(convert.String(rand.Int63())))
	return nil
}
