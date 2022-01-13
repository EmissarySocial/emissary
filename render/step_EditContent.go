package render

import (
	"io"
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/nebula"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	filename       string
	contentLibrary *nebula.Library
}

func NewStepEditContent(contentLibrary *nebula.Library, stepInfo datatype.Map) StepEditContent {

	filename := first.String(stepInfo.GetString("file"), stepInfo.GetString("actionId"))

	return StepEditContent{
		contentLibrary: contentLibrary,
		filename:       filename,
	}
}

func (step StepEditContent) Get(buffer io.Writer, renderer Renderer) error {

	context := renderer.context()
	params := context.QueryParams()

	// Handle action popups
	if action := convert.String(params["action"]); action != "" {
		return step.modalAction(buffer, renderer)
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

	// UPLOADS: If present, inject the uploaded filename into the form post. (One attachment per content item)
	if attachments := uploadedFiles(renderer.factory(), renderer.context(), renderer.objectID()); len(attachments) > 0 {
		body["file"] = "/" + renderer.Token() + "/attachments/" + attachments[0].Filename
		body["mimeType"] = attachments[0].MimeType()
	}

	// Try to execute the transaction
	container := getterSetter.GetContainer()
	changedID, err := container.Post(step.contentLibrary, body)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error executing content action")
	}

	// Write the updated content back into the object
	getterSetter.SetContainer(container)

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(object, "Content edited"); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error saving stream")
	}

	// If this is an "edit-item" action, then DON'T return HTML
	// to the browser because we might mess up the client-side state
	if body.GetString("type") == "update-item" {
		return renderer.context().NoContent(http.StatusOK)
	}

	// Otherwise, let's try to update the browser with some new content...

	// Close any modal dialogs that are open
	header := renderer.context().Response().Header()
	header.Set("HX-Trigger", "closeModal")
	header.Set("ChangedID", convert.String(changedID))
	header.Set("HX-Retarget", `.content-editor`)
	// header.Set("HX-Retarget", `[data-id="0"]`)
	// header.Set("HX-Retarget", `[data-id="`+convert.String(changedID)+`"]`)

	// Re-render ALL items, including the Sortable behavior
	result := nebula.Edit(step.contentLibrary, &container, renderer.URL())
	// Re-render JUST the updated item
	// b := html.New()
	// step.contentLibrary.Edit(b, &container, changedID, renderer.URL())
	// result := b.String()

	// Copy the result back to the client response
	if _, err := io.WriteString(buffer, result); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error writing to output buffer", result)
	}

	// Success!
	return nil
}

func (step StepEditContent) modalAction(buffer io.Writer, renderer Renderer) error {

	object := renderer.object()

	if getter, ok := object.(nebula.GetterSetter); ok {

		context := renderer.context()
		library := renderer.factory().ContentLibrary()
		content := getter.GetContainer()

		urlValues := context.Request().URL.Query()
		params := convert.MapOfInterface(urlValues)
		result := content.Get(library, params, renderer.URL())

		if result == "" {
			return derp.New(derp.CodeBadRequestError, "ghost.render.StepEditContent.Get", "No action modal available", params)
		}

		// Success!
		result = WrapModal(context.Response(), result)
		io.WriteString(buffer, result)
		return nil
	}

	// Generic error because someone done bad.
	return derp.NewInternalError("ghost.render.StepEditContent.Get", "Unable to create property panel.  Object is not a nebula.GetterSetter", object)

}
