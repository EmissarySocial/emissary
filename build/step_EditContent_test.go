package build

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/stretchr/testify/require"
)

// StepEditContent must reject a submission whose body exceeds the step's MaxLength BEFORE it
// reaches the database, bounding per-user storage and the size of posts amplified over federation.
// The rejection path halts in errorContentTooLong, which touches only object(), request(), and
// response() -- so a minimal stub builder (no factory/service/session) is enough.

// stubEditContentBuilder is a build.Builder exposing only the three methods that
// StepEditContent.Post reaches on the rejection path.
type stubEditContentBuilder struct {
	Builder
	stream *model.Stream
	req    *http.Request
	res    http.ResponseWriter
}

// object implements the Builder interface, returning this stub's Stream
func (b stubEditContentBuilder) object() data.Object { return b.stream }

// request implements the Builder interface, returning this stub's request
func (b stubEditContentBuilder) request() *http.Request { return b.req }

// response implements the Builder interface, returning this stub's response recorder
func (b stubEditContentBuilder) response() http.ResponseWriter { return b.res }

// editContentFormRequest builds a urlencoded POST carrying the given content in the "content" field.
func editContentFormRequest(content string) *http.Request {

	form := url.Values{"content": {content}}

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

// A form submission longer than MaxLength must halt the pipeline without saving.
func TestStepEditContent_Post_RejectsOversizeForm(t *testing.T) {

	stream := model.NewStream()

	builder := stubEditContentBuilder{
		stream: &stream,
		req:    editContentFormRequest(strings.Repeat("a", 50)),
		res:    httptest.NewRecorder(),
	}

	step := StepEditContent{Format: model.ContentFormatHTML, Fieldname: "content", MaxLength: 10}
	behavior := step.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "an oversize submission must halt the pipeline")
	require.Empty(t, stream.Content.Raw, "an oversize submission must not be stored")
}

// An EditorJS body (read directly from the request body) longer than MaxLength must also be
// rejected, and the LimitReader must keep an oversized body from being read without bound.
func TestStepEditContent_Post_RejectsOversizeEditorJS(t *testing.T) {

	stream := model.NewStream()

	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", 500)))

	builder := stubEditContentBuilder{
		stream: &stream,
		req:    request,
		res:    httptest.NewRecorder(),
	}

	step := StepEditContent{Format: model.ContentFormatEditorJS, MaxLength: 10}
	behavior := step.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "an oversize EditorJS submission must halt the pipeline")
	require.Empty(t, stream.Content.Raw, "an oversize EditorJS submission must not be stored")
}

// An empty EditorJS body must be rejected.  EditorJS always posts a JSON document, so an
// empty body means the body was consumed or never sent -- never that the author cleared the
// article -- and storing it would replace good content with nothing.
func TestStepEditContent_Post_RejectsEmptyEditorJS(t *testing.T) {

	stream := model.NewStream()
	stream.Content = model.Content{Format: model.ContentFormatEditorJS, Raw: `{"blocks":[]}`}

	builder := stubEditContentBuilder{
		stream: &stream,
		req:    httptest.NewRequest(http.MethodPost, "/", strings.NewReader("")),
		res:    httptest.NewRecorder(),
	}

	step := StepEditContent{Format: model.ContentFormatEditorJS, MaxLength: 1024}
	behavior := step.Post(builder, io.Discard)

	result := NewPipelineResult()
	behavior(&result)

	require.True(t, result.Halt, "an empty EditorJS submission must halt the pipeline")
	require.Error(t, result.Error, "an empty EditorJS submission must report an error")
	require.Equal(t, `{"blocks":[]}`, stream.Content.Raw, "an empty EditorJS submission must not overwrite the stored content")
}

// POST handlers run inside a MongoDB transaction whose callback is re-run whenever the driver
// sees a transient error, so StepEditContent.Post must be safe to execute more than once for a
// single request.  A second pass must read the same body as the first instead of finding a
// drained stream and treating it as empty content.
//
// The assertion rides on the oversize path because a successful save reaches builder.factory(),
// which this stub cannot provide: an oversize body halts in errorContentTooLong with no Error,
// while a drained body would halt with the empty-body Error instead.  The two are therefore
// distinguishable without a full builder.  The body must be shorter than the read bound
// (MaxLength*4+1 bytes) so that the first attempt consumes it completely -- a body long enough
// to trip the LimitReader would leave bytes behind and hide a missing rewind.
func TestStepEditContent_Post_RewindsBodyForTransactionRetry(t *testing.T) {

	stream := model.NewStream()

	builder := stubEditContentBuilder{
		stream: &stream,
		req:    httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", 20))),
		res:    httptest.NewRecorder(),
	}

	step := StepEditContent{Format: model.ContentFormatEditorJS, MaxLength: 10}

	// First attempt: reads the body to EOF, then halts because the content is too long
	firstResult := NewPipelineResult()
	step.Post(builder, io.Discard)(&firstResult)

	require.True(t, firstResult.Halt, "an oversize EditorJS submission must halt the pipeline")
	require.NoError(t, firstResult.Error, "the first attempt must halt on the too-long path")

	// Second attempt (the transaction retry): the body must still be readable
	secondResult := NewPipelineResult()
	step.Post(builder, io.Discard)(&secondResult)

	require.True(t, secondResult.Halt, "the retried submission must halt the pipeline")
	require.NoError(t, secondResult.Error, "a retry must re-read the body, not find it drained and empty")
	require.Empty(t, stream.Content.Raw, "an oversize EditorJS submission must not be stored")
}
