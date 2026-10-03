package mastodon

import (
	"encoding/base64"
	"errors"
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/stretchr/testify/require"
)

// fakeClient serves documents from a map, standing in for the network.
type fakeClient struct {
	documents map[string]map[string]any
}

// SetRootClient does nothing, because this client has no other clients behind it.
func (client *fakeClient) SetRootClient(streams.Client) {}

// Load returns the document stored under an address, or an error when there is none.
func (client *fakeClient) Load(uri string, _ ...any) (streams.Document, error) {

	if value, ok := client.documents[uri]; ok {
		return streams.NewDocument(value, streams.WithClient(client)), nil
	}

	return streams.NilDocument(), errors.New("not found: " + uri)
}

// Save does nothing, because a test's documents never change.
func (client *fakeClient) Save(streams.Document) error { return nil }

// Delete does nothing, because a test's documents never change.
func (client *fakeClient) Delete(string) error { return nil }

// newFollowersClient builds a three-actor followers collection split across two pages.
func newFollowersClient() *fakeClient {
	return &fakeClient{documents: map[string]map[string]any{
		"https://a.example/users/x/followers": {
			"id": "https://a.example/users/x/followers", "type": "OrderedCollection", "totalItems": 3,
			"first": "https://a.example/users/x/followers?page=1",
		},
		"https://a.example/users/x/followers?page=1": {
			"id": "https://a.example/users/x/followers?page=1", "type": "OrderedCollectionPage",
			"orderedItems": []any{"https://b.example/users/one", "https://b.example/users/two"},
			"next":         "https://a.example/users/x/followers?page=2",
		},
		"https://a.example/users/x/followers?page=2": {
			"id": "https://a.example/users/x/followers?page=2", "type": "OrderedCollectionPage",
			"orderedItems": []any{"https://b.example/users/three"},
		},
	}}
}

// TestCollectActorURLs_ReadsEveryPageWhenTheLimitAllows confirms the walk follows "next" until the collection ends.
func TestCollectActorURLs_ReadsEveryPageWhenTheLimitAllows(t *testing.T) {

	client := newFollowersClient()
	collection, err := client.Load("https://a.example/users/x/followers")
	require.NoError(t, err)

	urls, next := collectActorURLs(client, collection, "", 20)

	require.Equal(t, []string{"https://b.example/users/one", "https://b.example/users/two", "https://b.example/users/three"}, urls)
	require.Equal(t, "", next, "an exhausted collection has no next page")
}

// TestCollectActorURLs_StopsAtTheLimitAndHandsBackACursor confirms the second page is left for the next request,
// and that resuming from the cursor picks up exactly there.
func TestCollectActorURLs_StopsAtTheLimitAndHandsBackACursor(t *testing.T) {

	client := newFollowersClient()
	collection, err := client.Load("https://a.example/users/x/followers")
	require.NoError(t, err)

	urls, next := collectActorURLs(client, collection, "", 2)

	require.Equal(t, []string{"https://b.example/users/one", "https://b.example/users/two"}, urls)
	require.Equal(t, "https://a.example/users/x/followers?page=2", next)

	cursor := base64.RawURLEncoding.EncodeToString([]byte(next))
	urls, next = collectActorURLs(client, collection, cursor, 2)

	require.Equal(t, []string{"https://b.example/users/three"}, urls)
	require.Equal(t, "", next)
}

// TestCollectActorURLs_RefusesACursorOnAnotherHost confirms a made-up cursor cannot send the server elsewhere.
func TestCollectActorURLs_RefusesACursorOnAnotherHost(t *testing.T) {

	client := newFollowersClient()
	collection, err := client.Load("https://a.example/users/x/followers")
	require.NoError(t, err)

	elsewhere := base64.RawURLEncoding.EncodeToString([]byte("https://evil.example/anything"))
	urls, next := collectActorURLs(client, collection, elsewhere, 20)

	require.Empty(t, urls)
	require.Equal(t, "", next)

	urls, _ = collectActorURLs(client, collection, "!!not-base64!!", 20)
	require.Empty(t, urls)
}

// TestCollectActorURLs_ReadsItemsHeldInsideTheCollection covers a collection that lists its items directly.
func TestCollectActorURLs_ReadsItemsHeldInsideTheCollection(t *testing.T) {

	client := &fakeClient{documents: map[string]map[string]any{
		"https://a.example/users/x/following": {
			"id": "https://a.example/users/x/following", "type": "OrderedCollection",
			"orderedItems": []any{map[string]any{"id": "https://b.example/users/one", "type": "Person"}},
		},
	}}

	collection, err := client.Load("https://a.example/users/x/following")
	require.NoError(t, err)

	urls, next := collectActorURLs(client, collection, "", 20)

	require.Equal(t, []string{"https://b.example/users/one"}, urls)
	require.Equal(t, "", next)
}
