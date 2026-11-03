package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestCanonicalProfileURL_KeepsQuery pins that forwarding a profile request from its UserID to
// its username keeps the query string.  A collection's link redirects to its owner's
// /@{userId}?tab={collectionId}, and without the query the profile opened on its first tab.
func TestCanonicalProfileURL_KeepsQuery(t *testing.T) {

	require.Equal(t, "/@ada?tab=6ad1337b549f00286c6bb5e0", canonicalProfileURL("ada", "", "tab=6ad1337b549f00286c6bb5e0"))
	require.Equal(t, "/@ada/about?tab=about&x=1", canonicalProfileURL("ada", "about", "tab=about&x=1"))
}

// TestCanonicalProfileURL_NoQuery pins the forward of a request that has no query string, which
// must not gain a stray "?"
func TestCanonicalProfileURL_NoQuery(t *testing.T) {

	require.Equal(t, "/@ada", canonicalProfileURL("ada", "", ""))
	require.Equal(t, "/@ada/about", canonicalProfileURL("ada", "about", ""))
}
