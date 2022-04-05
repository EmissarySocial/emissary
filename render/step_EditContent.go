package render

import (
	"io"
	"net/http"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/nebula"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename string
}

func (step StepEditContent) Get(renderer Renderer, buffer io.Writer) error {

	context := renderer.context()
	params := context.QueryParams()

	// Handle action popups
	if action := convert.String(params["action"]); action != "" {
		return step.modalAction(buffer, renderer)
	}

	if err := renderer.executeTemplate(buffer, step.Filename, renderer); err != nil {
		return derp.Wrap(err, "whisper.render.StepEditContent.Get", "Error executing template")
	}

	return nil
}

func (step StepEditContent) Post(renderer Renderer, buffer io.Writer) error {

	factory := renderer.factory()
	object := renderer.object()
	getterSetter, ok := object.(nebula.GetterSetter)

	if !ok {
		return derp.NewInternalError("whisper.render.StepEditContent.Post", "Bad configuration - object does not have content to edit", renderer.object())
	}

	// Try to read the request body
	body := datatype.Map{}

	if err := renderer.context().Bind(&body); err != nil {
		return derp.Wrap(err, "whisper.render.StepEditContent.Post", "Error binding data")
	}

	// UPLOADS: If present, inject the uploaded filename into the form post. (One attachment per content item)
	if attachments := uploadedFiles(factory, renderer.context(), renderer.objectID()); len(attachments) > 0 {
		body["file"] = "/" + renderer.Token() + "/attachments/" + attachments[0].Filename
		body["mimeType"] = attachments[0].MimeType()
	}

	// Try to execute the transaction
	contentLibrary := factory.ContentLibrary()
	container := getterSetter.GetContainer()
	changedID, err := container.Post(contentLibrary, body)

	if err != nil {
		return derp.Wrap(err, "whisper.render.StepEditContent.Post", "Error executing content action")
	}

	// Write the updated content back into the object
	getterSetter.SetContainer(container)

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(object, "Content edited"); err != nil {
		return derp.Wrap(err, "whisper.render.StepEditContent.Post", "Error saving stream")
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

	// Re-render ALL items, including the Sortable behavior
	result := nebula.Edit(contentLibrary, &container, renderer.URL())

	// Copy the result back to the client response
	if _, err := io.WriteString(buffer, result); err != nil {
		return derp.Wrap(err, "whisper.render.StepEditContent.Post", "Error writing to output buffer", result)
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
			return derp.New(derp.CodeBadRequestError, "whisper.render.StepEditContent.Get", "No action modal available", params)
		}

		// Success!
		result = WrapModal(context.Response(), result)
		io.WriteString(buffer, result)
		return nil
	}

	// Generic error because someone done bad.
	return derp.NewInternalError("whisper.render.StepEditContent.Get", "Unable to create property panel.  Object is not a nebula.GetterSetter", object)

}
