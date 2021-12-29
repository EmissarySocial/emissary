package render

import (
	"io"

	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/nebula"
	"github.com/benpate/nebula/transaction"
)

// StepEditContent represents an action-step that can edit/update nebula.Container in a streamDraft.
type StepEditContent struct {
	filename string
}

func NewStepEditContent(stepInfo datatype.Map) StepEditContent {

	filename := first.String(stepInfo.GetString("file"), stepInfo.GetString("actionId"))

	return StepEditContent{
		filename: filename,
	}
}

func (step StepEditContent) Get(buffer io.Writer, renderer Renderer) error {

	// Handle transaction popups
	if transaction := renderer.context().QueryParam("transaction"); transaction != "" {
		return nil
	}

	if err := renderer.executeTemplate(buffer, step.filename, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Get", "Error executing template")
	}

	return nil
}

func (step StepEditContent) Post(buffer io.Writer, renderer Renderer) error {

	object := renderer.object()
	getterSetter, ok := object.(nebula.GetterSetter)

	if !ok {
		return derp.NewInternalError("ghost.render.StepEditContent.Post", "Bad configuration - object does not have content to edit", renderer.object())
	}

	// Try to read the request body
	body := datatype.Map{}

	if err := renderer.context().Bind(&body); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error binding data")
	}

	// Try to parse the request body as a transaction
	txn, err := transaction.Parse(body)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error parsing transaction", body)
	}

	// Try to execute the transaction

	c := getterSetter.GetContainer()
	if _, err := txn.Execute(&c); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error executing transaction", txn)
	}

	// Write the updated content back into the object
	getterSetter.SetContainer(c)

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(object, "Content edited: "+txn.Description()); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error saving stream")
	}

	// Close any modal dialogs that are open
	header := renderer.context().Response().Header()
	header.Set("HX-Trigger", "closeModal")

	// Rewrite the body to the client.
	// TODO: Perhaps this can be more efficient in the future
	if err := renderer.executeTemplate(buffer, step.filename, renderer); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error executing template")
	}

	// Return response to caller
	return nil
}
