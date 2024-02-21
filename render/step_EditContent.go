package render

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// StepEditContent represents an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename string
	Format   string
}

func (step StepEditContent) Get(renderer Renderer, buffer io.Writer) PipelineBehavior {

	if err := renderer.executeTemplate(buffer, step.Filename, renderer); err != nil {
		return Halt().WithError(derp.Wrap(err, "render.StepEditContent.Get", "Error executing template"))
	}

	return nil
}

func (step StepEditContent) Post(renderer Renderer, _ io.Writer) PipelineBehavior {

	var rawContent string

	// Require that we're working with a Stream
	stream, ok := renderer.object().(*model.Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError("render.StepEditContent.Post", "step: EditContent can only be used on a Stream"))
	}

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case model.ContentFormatEditorJS:
		var buffer bytes.Buffer

		if _, err := io.Copy(&buffer, renderer.request().Body); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepEditContent.Post", "Error reading request data"))
		}

		rawContent = buffer.String()

	// All other types are a Form post
	default:

		body := mapof.NewAny()
		if err := bind(renderer.request(), &body); err != nil {
			return Halt().WithError(derp.Wrap(err, "render.StepEditContent.Post", "Error parsing request data"))
		}

		rawContent, _ = body.GetStringOK("content")
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
		return Halt().WithError(derp.Wrap(err, "render.StepEditContent.Post", "Error saving stream"))
	}

	// Success!
	return nil
}
