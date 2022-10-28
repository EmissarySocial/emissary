package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename string
}

func (step StepEditContent) Get(renderer Renderer, buffer io.Writer) error {

	if err := renderer.executeTemplate(buffer, step.Filename, renderer); err != nil {
		return derp.Wrap(err, "render.StepEditContent.Get", "Error executing template")
	}

	return nil
}

func (step StepEditContent) UseGlobalWrapper() bool {
	return true
}

func (step StepEditContent) Post(renderer Renderer) error {

	context := renderer.context()
	factory := renderer.factory()
	stream := renderer.object().(*model.Stream)

	// Try to read the request body
	var content bytes.Buffer

	if _, err := io.Copy(&content, context.Request().Body); err != nil {
		return derp.Wrap(err, "render.StepEditContent.Post", "Error reading request data")
	}

	// Try to generate HTML from the EditorJS JSON
	editorjs := factory.EditorJS()
	html, err := editorjs.GenerateHTML(content.String())

	if err != nil {
		return derp.Wrap(err, "render.StepEditContent.Post", "Error converting EditorJS to HTML")
	}

	stream.Content = model.Content{
		Raw:  content.String(),
		HTML: html,
	}

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(stream, "Content edited"); err != nil {
		return derp.Wrap(err, "render.StepEditContent.Post", "Error saving stream")
	}

	// Success!
	return nil
}
