package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/maps"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename string
	Format   string
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

	var rawContent string

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case "EDITORJS":
		var buffer bytes.Buffer

		if _, err := io.Copy(&buffer, context.Request().Body); err != nil {
			return derp.Wrap(err, "render.StepEditContent.Post", "Error reading request data")
		}

		rawContent = buffer.String()

	// All other types are a Form post
	default:

		body := maps.New()
		if err := context.Bind(&body); err != nil {
			return derp.Wrap(err, "render.StepEditContent.Post", "Error parsing request data")
		}

		rawContent = body.GetString("content")
	}

	// Create a new Content object from the request body
	factory := renderer.factory()
	contentService := factory.Content()
	content := contentService.New(step.Format, rawContent)

	// Put the content into the stream
	stream := renderer.object().(*model.Stream)
	stream.Content = content

	// Try to save the object back to the database
	if err := renderer.service().ObjectSave(stream, "Content edited"); err != nil {
		return derp.Wrap(err, "render.StepEditContent.Post", "Error saving stream")
	}

	// Success!
	return nil
}
