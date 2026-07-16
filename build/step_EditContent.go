package build

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/html"
)

// StepEditContent is a Step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename       string
	Fieldname      string
	Format         string
	RequireContent bool
}

func (step StepEditContent) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditContent.Get"

	if step.Filename != "" {
		if err := builder.execute(buffer, step.Filename, builder); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Executing template"))
		}
	}

	return nil
}

func (step StepEditContent) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditContent.Post"

	var rawContent string

	// Require that we're working with a Stream
	stream, ok := builder.object().(*model.Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "step: EditContent can only be used on a Stream"))
	}

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case model.ContentFormatEditorJS:
		var buffer bytes.Buffer

		if _, err := io.Copy(&buffer, builder.request().Body); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Reading request data"))
		}

		rawContent = buffer.String()

	// All other types are a Form post
	default:

		value, err := formdata.Parse(builder.request())

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Parsing request data"))
		}

		rawContent = value.Get(step.Fieldname)
	}

	// RULE: If content is required, then reject empty submissions before saving.
	// ToText strips markup so that visually-empty content (e.g. "<br>" from a
	// contenteditable field) is also treated as empty.
	if step.RequireContent {
		if strings.TrimSpace(html.ToText(rawContent)) == "" {
			return step.errorEmptyContent(builder)
		}
	}

	// Set the new Content value in the Stream
	contentService := builder.factory().Content()
	stream.Content = contentService.New(step.Format, rawContent)

	// Try to save the object back to the database
	if err := builder.service().ObjectSave(builder.session(), stream, "Content edited"); err != nil {
		return Halt().WithError(derp.Wrap(err, location, "Saving stream"))
	}

	// Success!
	return nil
}

// errorEmptyContent writes an inline error message back to the compose form when
// content is required but the submission is empty. It halts the pipeline so that
// nothing is saved or published.
func (step StepEditContent) errorEmptyContent(builder Builder) PipelineBehavior {

	const location = "build.StepEditContent.errorEmptyContent"

	response := builder.response()
	response.Header().Set("HX-Reswap", "innerHTML")
	response.Header().Set("HX-Retarget", "#outbox-message-error")
	response.WriteHeader(http.StatusOK)

	if _, err := response.Write([]byte(`<span class="text-red">Please write something before posting.</span>`)); err != nil {
		derp.Report(derp.Wrap(err, location, "Writing error message to response"))
	}

	return Halt().AsFullPage()
}
