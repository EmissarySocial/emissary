package activitypub

import (
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// mlsCreate builds the canonical qualifying activity: a Create of an inline, non-public MLS
// message. Tests mutate one knob at a time to prove each condition is load-bearing.
func mlsCreate() mapof.Any {
	return mapof.Any{
		vocab.PropertyType:  vocab.ActivityTypeCreate,
		vocab.PropertyActor: "https://example.com/@alice",
		vocab.PropertyTo:    []any{"https://example.com/@bob"},
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType:      "Note",
			vocab.PropertyMediaType: vocab.MediaTypeMLS,
			vocab.PropertyContent:   "SGVsbG8sIGNpcGhlcnRleHQh",
		},
	}
}

// TestIsMLSCreate verifies which document shapes qualify as an MLS message, and which do not
func TestIsMLSCreate(t *testing.T) {

	// The canonical shape qualifies
	require.True(t, IsMLSCreate(streams.NewDocument(mlsCreate())))

	// Update does not qualify: all MLS arrives as Create
	update := mlsCreate()
	update[vocab.PropertyType] = vocab.ActivityTypeUpdate
	require.False(t, IsMLSCreate(streams.NewDocument(update)))

	// A link-shaped object does not qualify: inline maps only
	linked := mlsCreate()
	linked[vocab.PropertyObject] = "https://example.com/notes/1"
	require.False(t, IsMLSCreate(streams.NewDocument(linked)))

	// A non-MLS mediaType does not qualify
	plaintext := mlsCreate()
	plaintext[vocab.PropertyObject].(map[string]any)[vocab.PropertyMediaType] = "text/html"
	require.False(t, IsMLSCreate(streams.NewDocument(plaintext)))

	// A missing mediaType does not qualify
	untyped := mlsCreate()
	delete(untyped[vocab.PropertyObject].(map[string]any), vocab.PropertyMediaType)
	require.False(t, IsMLSCreate(streams.NewDocument(untyped)))

	// Public addressing does not qualify: real group ciphertext is never public
	public := mlsCreate()
	public[vocab.PropertyTo] = []any{vocab.NamespaceActivityStreamsPublic}
	require.False(t, IsMLSCreate(streams.NewDocument(public)))
}
