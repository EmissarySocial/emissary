package assanitizer

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// fakeInner is a streams.Client that returns a canned document and records what it was asked to
// Save and Delete, so tests can assert what passed through the sanitizer.
type fakeInner struct {
	document streams.Document
	loadErr  error
	saved    []streams.Document
	deleted  []string
}

func (fake *fakeInner) SetRootClient(streams.Client) {}

func (fake *fakeInner) Load(uri string, options ...any) (streams.Document, error) {
	return fake.document, fake.loadErr
}

func (fake *fakeInner) Save(document streams.Document) error {
	fake.saved = append(fake.saved, document)
	return nil
}

func (fake *fakeInner) Delete(documentID string) error {
	fake.deleted = append(fake.deleted, documentID)
	return nil
}

func TestClient_LoadStrips(t *testing.T) {

	inner := &fakeInner{
		document: streams.NewDocument(map[string]any{
			vocab.PropertyID:  "https://example.com/note/1",
			"emissary:labels": "forged",
			"object": map[string]any{
				"type":            "Note",
				"emissary:labels": "forged",
			},
		}),
	}

	client := New(inner, "emissary:")
	document, err := client.Load("https://example.com/note/1")

	require.NoError(t, err)
	require.Equal(t,
		map[string]any{
			vocab.PropertyID: "https://example.com/note/1",
			"object": map[string]any{
				"type": "Note",
			},
		},
		map[string]any(document.Map()))
}

func TestClient_LoadErrorPassesThrough(t *testing.T) {

	inner := &fakeInner{
		document: streams.NilDocument(),
		loadErr:  derp.Internal("test", "the tubes are clogged"),
	}

	client := New(inner, "emissary:")
	_, err := client.Load("https://example.com/note/1")

	require.Error(t, err)
}

func TestClient_LoadStringDocument(t *testing.T) {

	// A document whose value is a bare string is left alone (nothing to strip, nothing to panic on)
	inner := &fakeInner{
		document: streams.NewDocument("https://example.com/note/1"),
	}

	client := New(inner, "emissary:")
	_, err := client.Load("https://example.com/note/1")

	require.NoError(t, err)
}

func TestClient_SaveStrips(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, "emissary:")

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:  "https://example.com/note/2",
		"emissary:labels": "forged",
	})

	require.NoError(t, client.Save(document))
	require.Len(t, inner.saved, 1)

	// The inner client only ever saw the clean document
	require.Equal(t,
		map[string]any{vocab.PropertyID: "https://example.com/note/2"},
		map[string]any(inner.saved[0].Map()))
}

func TestClient_DeletePassesThrough(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, "emissary:")

	require.NoError(t, client.Delete("https://example.com/note/3"))
	require.Equal(t, []string{"https://example.com/note/3"}, inner.deleted)
}
