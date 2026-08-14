package service

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/tools/ascache"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// signatureKeyID is the keyId used by the fixtures below.
const signatureKeyID = "https://remote.example/@alice#main-key"

// fakeKeyStore stands in for the ActivityStream client stack: it hands out a PEM, records how it was
// asked, and can "rotate" its key between calls.
type fakeKeyStore struct {
	publicKeyPEM string // Key served to the next caller
	rotateTo     string // If set, the key the SECOND and later loads serve, simulating a remote rotation
	loads        int    // Number of times the loader ran
	forcedLoads  int    // Number of loads that carried WithWriteOnly, i.e. bypassed the cached copy
	err          error  // If set, every load fails with this error
}

// load implements publicKeyLoader.
func (store *fakeKeyStore) load(keyID string, options ...any) (streams.Document, error) {

	store.loads++

	// Recording the MODE is the point: a refresh that did not force a reload would be answered by the
	// same cached key that just failed, and rotation would silently stop being repaired.
	for _, option := range options {
		if loadOption, ok := option.(ascache.LoadOption); ok {
			config := ascache.NewLoadConfig(loadOption)
			if config.IsWriteOnly() {
				store.forcedLoads++
			}
		}
	}

	if store.err != nil {
		return streams.NilDocument(), store.err
	}

	// The rotation lands AFTER the first answer, so the stale key is the one the first pass sees.
	publicKeyPEM := store.publicKeyPEM

	if store.rotateTo != "" {
		store.publicKeyPEM = store.rotateTo
		store.rotateTo = ""
	}

	document := streams.NewDocument(
		mapof.Any{
			"id":           keyID,
			"owner":        "https://remote.example/@alice",
			"publicKeyPem": publicKeyPEM,
		},
	)

	return document, nil
}

// finders returns the primary/refresh pair exactly as the inbox and VerifySignature compose them.
func (store *fakeKeyStore) finders() (sigs.PublicKeyFinder, sigs.PublicKeyFinder) {

	find := func(keyID string) (string, error) {
		return findPublicKey(store.load, keyID)
	}

	refresh := func(keyID string) (string, error) {
		return refreshPublicKey(store.load, keyID)
	}

	return find, refresh
}

// newSignedRequest returns a GET request signed with a fresh key pair, plus that key's PEM.
func newSignedRequest(t *testing.T) (*http.Request, string) {

	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)
	require.NoError(t, sigs.Sign(request, signatureKeyID, privateKey))

	return request, sigs.EncodePublicPEM(privateKey)
}

// TestFindPublicKey_ReturnsPEM is the plain case: the finder extracts the certificate that the loaded
// Actor document publishes.
func TestFindPublicKey_ReturnsPEM(t *testing.T) {

	store := &fakeKeyStore{publicKeyPEM: "PEM-ORIGINAL"}

	publicKeyPEM, err := findPublicKey(store.load, signatureKeyID)

	require.NoError(t, err)
	require.Equal(t, "PEM-ORIGINAL", publicKeyPEM)
	require.Equal(t, 1, store.loads)
	require.Equal(t, 0, store.forcedLoads, "the normal finder must be allowed to use the cache")
}

// TestRefreshPublicKey_ForcesReload pins the one thing that separates the refresh from the normal
// finder.  Without WithWriteOnly it would be answered from the cache, by the very key that just failed.
func TestRefreshPublicKey_ForcesReload(t *testing.T) {

	store := &fakeKeyStore{publicKeyPEM: "PEM-ROTATED"}

	publicKeyPEM, err := refreshPublicKey(store.load, signatureKeyID)

	require.NoError(t, err)
	require.Equal(t, "PEM-ROTATED", publicKeyPEM)
	require.Equal(t, 1, store.forcedLoads)
}

// TestRefreshPublicKey_ReportsFailure confirms a failed reload is surfaced rather than swallowed.
// sigs.Verify is what decides to drop it in favor of the original error.
func TestRefreshPublicKey_ReportsFailure(t *testing.T) {

	store := &fakeKeyStore{err: derp.Internal("test", "Remote host unreachable")}

	_, err := refreshPublicKey(store.load, signatureKeyID)

	require.Error(t, err)
}

// TestVerifySignature_CachedKeyCostsNothing is the burst-collapse case, stated directly: a signature
// that verifies on the first pass produces exactly one lookup and no forced reload.
func TestVerifySignature_CachedKeyCostsNothing(t *testing.T) {

	request, publicKeyPEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: publicKeyPEM}

	find, refresh := store.finders()
	signature, err := sigs.Verify(request, find, sigs.WithRefreshKey(refresh))

	require.NoError(t, err)
	require.Equal(t, "https://remote.example/@alice", signature.ActorID())
	require.Equal(t, 1, store.loads)
	require.Equal(t, 0, store.forcedLoads)
}

// TestVerifySignature_RotationRepairedOnFirstFailure is the property that motivated failure-driven
// refresh over a flat TTL: a peer that rotates its key is verified on the very first delivery that
// fails, with no window of rejected traffic.
func TestVerifySignature_RotationRepairedOnFirstFailure(t *testing.T) {

	request, rotatedPEM := newSignedRequest(t)

	// The cache still holds the peer's PREVIOUS key, so the first pass cannot verify.  The forced
	// reload that follows finds the key the peer is signing with now.
	_, stalePEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: stalePEM, rotateTo: rotatedPEM}

	find, refresh := store.finders()
	signature, err := sigs.Verify(request, find, sigs.WithRefreshKey(refresh))

	require.NoError(t, err)
	require.Equal(t, "https://remote.example/@alice", signature.ActorID())
	require.Equal(t, 2, store.loads, "one normal read, then one forced reload")
	require.Equal(t, 1, store.forcedLoads)
}

// TestVerifySignature_UnloadableKeyIsNotRetried covers the peer we cannot reach: the first pass
// already failed to produce a key, so there is nothing for a refresh to improve on.
func TestVerifySignature_UnloadableKeyIsNotRetried(t *testing.T) {

	request, _ := newSignedRequest(t)
	store := &fakeKeyStore{err: derp.Internal("test", "Remote host unreachable")}

	find, refresh := store.finders()
	_, err := sigs.Verify(request, find, sigs.WithRefreshKey(refresh))

	require.Error(t, err)
	require.Equal(t, 1, store.loads)
	require.Equal(t, 0, store.forcedLoads)
}
