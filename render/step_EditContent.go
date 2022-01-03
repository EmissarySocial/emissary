package render

import (
	"io"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/first"
	"github.com/benpate/html"
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

	// Handle transaction popups
	if transaction := convert.String(params["prop"]); transaction != "" {

		object := renderer.object()

		if getter, ok := object.(nebula.GetterSetter); ok {
			content := getter.GetContainer()
			itemID := convert.Int(params["itemId"])

			// Get the property panel from Nebula
			result, err := nebula.Prop(step.contentLibrary, &content, itemID, context.Request().Referer(), params)

			if err != nil {
				return derp.Wrap(err, "ghost.render.StepEditContent.Get", "Error rendering property panel", params)
			}

			// Success!
			result = WrapModalWithCloseButton(context.Response(), result)
			io.WriteString(buffer, result)
			return nil
		}

		// Generic error because someone done bad.
		return derp.NewInternalError("ghost.render.StepEditContent.Get", "Unable to create property panel", params)
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

	// Try to execute the transaction
	container := getterSetter.GetContainer()
	updatedID, err := container.Execute(step.contentLibrary, body)

	if err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error executing content action")
	}

	// Write the updated content back into the object
	getterSetter.SetContainer(container)

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(object, "Content edited"); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error saving stream")
	}

	// Close any modal dialogs that are open
	header := renderer.context().Response().Header()
	header.Set("HX-Trigger", "closeModal")

	// Re-render JUST the updated item
	b := html.New()
	b.Div().Data("hx-swap-oob", "itemId-"+convert.String(updatedID)).EndBracket()
	step.contentLibrary.Edit(b, &container, updatedID, renderer.URL())
	b.CloseAll()

	// Copy the result back to the client response
	if _, err := io.WriteString(buffer, b.String()); err != nil {
		return derp.Wrap(err, "ghost.render.StepEditContent.Post", "Error writing to output buffer", b.String())
	}

	// Success!
	return nil
}
