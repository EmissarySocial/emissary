package service

import (
	"testing"
	"text/template"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/templatemap"
	"github.com/stretchr/testify/require"
)

// newSearchableStream returns a Stream service whose test collection is searchable, and a
// Stream on it that anyone may view
func newSearchableStream(t *testing.T) (*Stream, model.Stream) {

	t.Helper()

	service := newStreamTokenService()

	name, err := template.New("name").Parse("{{.Label}}")
	require.NoError(t, err)

	collection := service.templateService.templates["test-collection"]
	collection.SearchOptions = templatemap.Map{"name": name}
	service.templateService.templates["test-collection"] = collection

	stream := newTokenStream("")
	stream.URL = "https://example.test/article"
	stream.Label = "An Article"
	stream.DefaultAllow = model.NewAnonymousPermissions()

	return service, stream
}

// TestStream_SearchResult_NeedsNoOutbox confirms that a Stream anyone may view is searchable
// when it is not in the outbox, as an article is, and whatever its dates say
func TestStream_SearchResult_NeedsNoOutbox(t *testing.T) {

	service, stream := newSearchableStream(t)
	require.False(t, stream.IsPublished())

	result := service.SearchResult(&stream)
	require.Zero(t, result.DeleteDate)
	require.Equal(t, "An Article", result.Name)
	require.True(t, result.Local)
}

// TestStream_SearchResult_InTheOutbox confirms that a Stream in the outbox is searchable too
func TestStream_SearchResult_InTheOutbox(t *testing.T) {

	service, stream := newSearchableStream(t)
	stream.IsOutbox.Set(true)

	result := service.SearchResult(&stream)
	require.Zero(t, result.DeleteDate)
	require.Equal(t, "An Article", result.Name)
}

// TestStream_SearchResult_RestrictedView confirms that a Stream is not searchable unless anyone may view it
func TestStream_SearchResult_RestrictedView(t *testing.T) {

	service, stream := newSearchableStream(t)
	stream.IsOutbox.Set(true)
	stream.DefaultAllow = model.NewPermissions()

	result := service.SearchResult(&stream)
	require.NotZero(t, result.DeleteDate)
	require.Equal(t, stream.URL, result.URL)
}

// TestStream_SearchResult_Deleted confirms that a deleted Stream is removed from the index
func TestStream_SearchResult_Deleted(t *testing.T) {

	service, stream := newSearchableStream(t)
	stream.IsOutbox.Set(true)
	stream.DeleteDate = 100

	result := service.SearchResult(&stream)
	require.NotZero(t, result.DeleteDate)
	require.Equal(t, stream.URL, result.URL)
}
