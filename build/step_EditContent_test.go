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
