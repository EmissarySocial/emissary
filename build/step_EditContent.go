package build

import (
	"bytes"
	"io"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// StepEditContent is an action-step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename  string
	Fieldname string
	Format    string
}

func (step StepEditContent) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	if step.Filename != "" {
		if err := builder.execute(buffer, step.Filename, builder); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepEditContent.Get", "Error executing template"))
		}
	}

	return nil
}

func (step StepEditContent) Post(builder Builder, _ io.Writer) PipelineBehavior {

	var rawContent string

	// Require that we're working with a Stream
	stream, ok := builder.object().(*model.Stream)

	if !ok {
		return Halt().WithError(derp.NewInternalError("build.StepEditContent.Post", "step: EditContent can only be used on a Stream"))
	}

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case model.ContentFormatEditorJS:
		var buffer bytes.Buffer

		if _, err := io.Copy(&buffer, builder.request().Body); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepEditContent.Post", "Error reading request data"))
		}

		rawContent = buffer.String()

	// All other types are a Form post
	default:

		body := mapof.NewAny()
		if err := bind(builder.request(), &body); err != nil {
			return Halt().WithError(derp.Wrap(err, "build.StepEditContent.Post", "Error parsing request data"))
		}

		rawContent = body.GetString(step.Fieldname)
	}

	// Set the new Content value in the Stream
	contentService := builder.factory().Content()
	stream.Content = contentService.New(step.Format, rawContent)

	// Try to save the object back to the database
	if err := builder.service().ObjectSave(stream, "Content edited"); err != nil {
		return Halt().WithError(derp.Wrap(err, "build.StepEditContent.Post", "Error saving stream"))
	}

	// Success!
	return nil
}
