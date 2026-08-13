package service

import (
	"testing"

	"github.com/benpate/derp"
	"github.com/stretchr/testify/require"
)

// TestPublicKeyFinder_Refreshable pins the gate that decides whether a failed verification is worth a
// second attempt.
func TestPublicKeyFinder_Refreshable(t *testing.T) {

	// A cached key may have been rotated away from
	t.Run("cached key is refreshable", func(t *testing.T) {
		finder := &publicKeyFinder{publicKeyPEM: "PEM", fromCache: true}
		require.True(t, finder.isRefreshable())
		require.False(t, finder.notRefreshable())
	})

	// A key straight from the origin is already the newest one there is
	t.Run("fresh key is not refreshable", func(t *testing.T) {
		finder := &publicKeyFinder{publicKeyPEM: "PEM", fromCache: false}
		require.True(t, finder.notRefreshable())
	})

	// A finder that never resolved a key has nothing to compare a reload against
	t.Run("unused finder is not refreshable", func(t *testing.T) {
		finder := &publicKeyFinder{fromCache: true}
		require.True(t, finder.notRefreshable())
	})
}

// TestPublicKeyFinder_Find confirms what the finder records on each outcome, since those three fields
// are the entire input to the refresh decision above.
func TestPublicKeyFinder_Find(t *testing.T) {

	// A cached answer records the PEM, the keyID, and the cache provenance
	t.Run("cached key is recorded", func(t *testing.T) {

		store := &fakeKeyStore{publicKeyPEM: "PEM-CACHED", fromCache: true}
		finder := newPublicKeyFinder(store.load)

		publicKeyPEM, err := finder.find(signatureKeyID)

		require.NoError(t, err)
		require.Equal(t, "PEM-CACHED", publicKeyPEM)
		require.Equal(t, signatureKeyID, finder.keyID)
		require.True(t, finder.isRefreshable())
	})

	// A fresh answer records the same PEM, but must NOT be marked refreshable
	t.Run("fresh key is recorded but not refreshable", func(t *testing.T) {

		store := &fakeKeyStore{publicKeyPEM: "PEM-FRESH", fromCache: false}
		finder := newPublicKeyFinder(store.load)

		_, err := finder.find(signatureKeyID)

		require.NoError(t, err)
		require.True(t, finder.notRefreshable())
	})

	// A FAILED load still records the keyID.  It must stay distinguishable from "never called", which
	// is what keeps a reload from being attempted against an empty keyID.
	t.Run("failed load still records the keyID", func(t *testing.T) {

		store := &fakeKeyStore{err: derp.Internal("test", "Remote host unreachable")}
		finder := newPublicKeyFinder(store.load)

		_, err := finder.find(signatureKeyID)

		require.Error(t, err)
		require.Equal(t, signatureKeyID, finder.keyID)
		require.Empty(t, finder.publicKeyPEM)
		require.True(t, finder.notRefreshable())
	})
}
