package asblock

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// fakeInner is a streams.Client that records the calls it receives, standing in for the network stack.
type fakeInner struct {
	loaded  []string
	saved   int
	deleted int
}

func (c *fakeInner) SetRootClient(streams.Client) {}

func (c *fakeInner) Load(uri string, _ ...any) (streams.Document, error) {
	c.loaded = append(c.loaded, uri)
	return streams.NewDocument(map[string]any{vocab.PropertyID: uri}), nil
}

func (c *fakeInner) Save(streams.Document) error { c.saved++; return nil }
func (c *fakeInner) Delete(string) error         { c.deleted++; return nil }

// A blocked origin is refused, and the network stack is never reached.
func TestClient_BlockedRefused(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(string) (bool, error) { return true, nil })

	document, err := client.Load("https://evil.example/post")

	require.NotNil(t, err)
	require.True(t, document.IsNil())
	require.Empty(t, inner.loaded)
}

// A clean origin passes through to the network stack.
func TestClient_CleanPassthrough(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(string) (bool, error) { return false, nil })

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.Equal(t, "https://good.example/post", document.ID())
	require.Equal(t, []string{"https://good.example/post"}, inner.loaded)
}

// A block-check error fails OPEN: the fetch proceeds rather than halting all federation.
func TestClient_ErrorFailsOpen(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(string) (bool, error) { return false, derp.Internal("test", "database down") })

	document, err := client.Load("https://good.example/post")

	require.Nil(t, err)
	require.Equal(t, "https://good.example/post", document.ID())
	require.Equal(t, []string{"https://good.example/post"}, inner.loaded)
}

// Save and Delete pass through unchanged -- blocking gates reads, not cache writes.
func TestClient_SaveDeletePassthrough(t *testing.T) {

	inner := &fakeInner{}
	client := New(inner, func(string) (bool, error) { return true, nil })

	require.Nil(t, client.Save(streams.NilDocument()))
	require.Nil(t, client.Delete("https://x.example/1"))
	require.Equal(t, 1, inner.saved)
	require.Equal(t, 1, inner.deleted)
}
