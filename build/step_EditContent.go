package build

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/formdata"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/html"
)

// editContentFallbackMaxLength bounds content (in runes -- every length limit in this
// codebase counts characters) when a StepEditContent is somehow built without a MaxLength.
// The normal configuration path (model/step.NewEditContent) always sets a limit; this only
// guards against a zero-valued struct, and mirrors that package's default.
const editContentFallbackMaxLength = 64 << 10 // 65,536 runes

// StepEditContent is a Step that can edit/update Container in a streamDraft.
type StepEditContent struct {
	Filename       string
	Fieldname      string
	Format         string
	MaxLength      int
	RequireContent bool
}

// Get renders this step during a GET request. Implements the Step interface.
func (step StepEditContent) Get(builder Builder, buffer io.Writer) PipelineBehavior {

	const location = "build.StepEditContent.Get"

	if step.Filename != "" {
		if err := builder.execute(buffer, step.Filename, builder); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Executing template"))
		}
	}

	return nil
}

// Post applies this step during a POST request. Implements the Step interface.
func (step StepEditContent) Post(builder Builder, _ io.Writer) PipelineBehavior {

	const location = "build.StepEditContent.Post"

	var rawContent string

	// Require that we're working with a Stream
	stream, ok := builder.object().(*model.Stream)

	if !ok {
		return Halt().WithError(derp.Internal(location, "step: EditContent can only be used on a Stream"))
	}

	// RULE: Determine the maximum content length (in runes) for this step.  NewEditContent
	// always populates MaxLength, but guard against a zero value so a mis-built step still
	// bounds the request instead of rejecting everything.
	maxLength := step.MaxLength

	if maxLength <= 0 {
		maxLength = editContentFallbackMaxLength
	}

	// Try to read the content from the request body
	switch step.Format {

	// EditorJS writes directly to the request body
	case model.ContentFormatEditorJS:
		var buffer bytes.Buffer

		// Bound the read so an oversized body cannot exhaust memory.  A rune is at most 4 UTF-8
		// bytes, so any body longer than (maxLength*4) bytes necessarily exceeds maxLength runes.
		// Read one extra byte so that hitting the limit is unambiguously "too long".
		maxBytes := int64(maxLength)*4 + 1

		if _, err := io.Copy(&buffer, io.LimitReader(builder.request().Body, maxBytes)); err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Reading request data"))
		}

		rawContent = buffer.String()

		// RULE: POST handlers run inside a MongoDB transaction, and the driver re-runs the
		// entire callback when a transient error is labeled on it -- a WriteConflict raised by
		// a second, concurrent write to the same StreamDraft is the common one.  An
		// http.Request.Body is a one-shot stream, so a retried attempt reads zero bytes and
		// would silently overwrite the Stream's content with an empty string.  Rewind the body
		// after each read so that every attempt sees the same payload.  The Form branch below
		// needs no equivalent because ParseForm caches its result on the Request.
		builder.request().Body = io.NopCloser(strings.NewReader(rawContent))

		// RULE: EditorJS always posts a JSON document, so an empty body means the body was
		// consumed or never sent -- never that the author cleared the article.  Reject it
		// rather than replacing good content with nothing.
		if rawContent == "" {
			return Halt().WithError(derp.BadRequest(location, "Empty request body"))
		}

	// All other types are a Form post
	default:

		value, err := formdata.Parse(builder.request())

		if err != nil {
			return Halt().WithError(derp.Wrap(err, location, "Parsing request data"))
		}

		rawContent = value.Get(step.Fieldname)
	}

	// RULE: Reject content that exceeds the configured maximum length.  This bounds
	// per-user storage growth and the size of posts amplified to followers over federation.
	// Length is measured in runes to match the schema-level cap in model.ContentSchema.
	if utf8.RuneCountInString(rawContent) > maxLength {
		return step.errorContentTooLong(builder, maxLength)
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

// errorContentTooLong writes an inline error message back to the compose form when the
// submission exceeds the maximum allowed length.  It mirrors errorEmptyContent: the
// pipeline halts so that nothing is saved or published, and the message is swapped into
// the compose form so an interactive author sees why the post was rejected.
func (step StepEditContent) errorContentTooLong(builder Builder, maxLength int) PipelineBehavior {

	const location = "build.StepEditContent.errorContentTooLong"

	response := builder.response()
	response.Header().Set("HX-Reswap", "innerHTML")
	response.Header().Set("HX-Retarget", "#outbox-message-error")
	response.WriteHeader(http.StatusOK)

	message := `<span class="text-red">Your post is too long. The maximum length is ` + strconv.Itoa(maxLength) + ` characters.</span>`

	if _, err := response.Write([]byte(message)); err != nil {
		derp.Report(derp.Wrap(err, location, "Writing error message to response"))
	}

	return Halt().AsFullPage()
}
