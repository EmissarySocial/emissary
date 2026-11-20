package asnormalizer

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

func TestHashedID(t *testing.T) {

	first := HashedID("https://example.com/notes/1")
	second := HashedID("https://example.com/notes/2")

	require.NotEmpty(t, first)
	require.NotEqual(t, first, second, "two documents must not share an element name")
	require.Equal(t, first, HashedID("https://example.com/notes/1"), "the same ID always hashes the same way")

	// The value is pinned, because cached rows and rendered pages both hold it
	require.Equal(t, "euhmr6t8thqcp", HashedID(""))
}

func TestStamp_Note(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:   "https://example.com/notes/1",
		vocab.PropertyType: vocab.ObjectTypeNote,
	})

	Stamp(&document)

	require.Equal(t, HashedID("https://example.com/notes/1"), document.Metadata.HashedID)
	require.Equal(t, vocab.DocumentCategoryObject, document.Metadata.DocumentCategory)
	require.Empty(t, document.Metadata.RelationType)
	require.Empty(t, document.Metadata.RelationHref)
}

func TestStamp_Reply(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:        "https://example.com/notes/2",
		vocab.PropertyType:      vocab.ObjectTypeNote,
		vocab.PropertyInReplyTo: "https://example.com/notes/1",
	})

	Stamp(&document)

	require.Equal(t, vocab.RelationTypeReply, document.Metadata.RelationType)
	require.Equal(t, "https://example.com/notes/1", document.Metadata.RelationHref)
	require.NotEqual(t, HashedID("https://example.com/notes/1"), document.Metadata.HashedID, "a reply is not named after its parent")
}

func TestStamp_Actor(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:   "https://example.com/@alice",
		vocab.PropertyType: vocab.ActorTypePerson,
	})

	Stamp(&document)

	require.Equal(t, vocab.DocumentCategoryActor, document.Metadata.DocumentCategory)
	require.Equal(t, HashedID("https://example.com/@alice"), document.Metadata.HashedID)
	require.Empty(t, document.Metadata.RelationType, "only an object records what it responds to")
}

func TestStamp_KeepsOtherMetadata(t *testing.T) {

	document := streams.NewDocument(map[string]any{
		vocab.PropertyID:   "https://example.com/notes/1",
		vocab.PropertyType: vocab.ObjectTypeNote,
	})
	document.Metadata.NoStore = true

	Stamp(&document)

	require.True(t, document.Metadata.NoStore, "Stamp writes only the fields it derives")
}

func TestStamp_EmptyDocument(t *testing.T) {

	// A document with no ID and no type must not panic
	document := streams.NilDocument()
	require.NotPanics(t, func() { Stamp(&document) })
}
