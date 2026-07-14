package model

import (
	"testing"

	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
)

// CollectionTypeForResponse maps Like/Dislike to their collection types, and everything else to "".
func TestCollectionTypeForResponse(t *testing.T) {

	require.Equal(t, CollectionTypeLikes, CollectionTypeForResponse(vocab.ActivityTypeLike))
	require.Equal(t, CollectionTypeDislikes, CollectionTypeForResponse(vocab.ActivityTypeDislike))
	require.Equal(t, CollectionTypeShares, CollectionTypeForResponse(vocab.ActivityTypeAnnounce))

	// Non-projected response types (and junk) return the empty string.
	require.Equal(t, "", CollectionTypeForResponse(""))
	require.Equal(t, "", CollectionTypeForResponse("Bookmark"))
}
