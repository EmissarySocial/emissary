package service

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// TestTagRelayName lowercases a tag and strips its #, keeping the characters tags.pub keeps
func TestTagRelayName(t *testing.T) {
	require.Equal(t, "cats", TagRelayName(" #Cats "))
	require.Equal(t, "game_dev", TagRelayName("Game_Dev"))
}

// TestRelayPost fetches an Announce's post from its origin, and a bare post by its own ID
func TestRelayPost(t *testing.T) {

	note := map[string]any{vocab.PropertyID: "https://remote.social/notes/1", vocab.PropertyType: vocab.ObjectTypeNote}
	client := annotationClient{document: note}

	announce := streams.NewDocument(map[string]any{
		vocab.PropertyID:     "https://tags.pub/announces/1",
		vocab.PropertyType:   vocab.ActivityTypeAnnounce,
		vocab.PropertyObject: "https://remote.social/notes/1",
	}, streams.WithClient(client))

	post, err := relayPost(announce)
	require.NoError(t, err)
	require.Equal(t, "https://remote.social/notes/1", post.ID())

	post, err = relayPost(streams.NewDocument(note, streams.WithClient(client)))
	require.NoError(t, err)
	require.Equal(t, "https://remote.social/notes/1", post.ID())
}
