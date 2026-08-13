package service

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"strings"
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

// fakeKeyStore stands in for the Actor cache: it hands out a PEM, records how many times it was
// asked, and marks its answers as cached or fresh.
type fakeKeyStore struct {
	publicKeyPEM string // Key served to the next caller
	rotateTo     string // If set, the key the SECOND and later loads serve, simulating a remote rotation
	fromCache    bool   // Whether answers are stamped as coming from the cache
	loads        int    // Number of times the loader ran (an outbound fetch, unless stamped cached)
	err          error  // If set, every load fails with this error
}

// load implements publicKeyLoader.
func (store *fakeKeyStore) load(keyID string, options ...any) (streams.Document, error) {

	store.loads++

	if store.err != nil {
		return streams.NilDocument(), store.err
	}

	// The rotation lands AFTER the first answer, so the cached key is the one the first pass sees.
	publicKeyPEM := store.publicKeyPEM

	if store.rotateTo != "" {
		store.publicKeyPEM = store.rotateTo
		store.rotateTo = ""
	}

	header := make(http.Header)

	if store.fromCache {
		header.Set(ascache.HeaderHannibalCache, "true")
	}

	document := streams.NewDocument(
		mapof.Any{
			"id":           keyID,
			"owner":        "https://remote.example/@alice",
			"publicKeyPem": publicKeyPEM,
		},
		streams.WithHTTPHeader(header),
	)

	return document, nil
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

// TestVerifySignature_CachedKeyCostsNothing is the burst-collapse case, stated directly: a signature
// that verifies against the cached key produces exactly one (cache-served) lookup and no retry.
func TestVerifySignature_CachedKeyCostsNothing(t *testing.T) {

	request, publicKeyPEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: publicKeyPEM, fromCache: true}

	signature, err := verifySignature(request, store.load)

	require.NoError(t, err)
	require.Equal(t, "https://remote.example/@alice", signature.ActorID())
	require.Equal(t, 1, store.loads)
}

// TestVerifySignature_RotationRepairedOnFirstFailure is the property that motivated failure-driven
// refresh over a flat TTL: a peer that rotates its key is verified on the very first delivery that
// fails, with no window of rejected traffic.
func TestVerifySignature_RotationRepairedOnFirstFailure(t *testing.T) {

	request, rotatedPEM := newSignedRequest(t)

	// The cache still holds the peer's PREVIOUS key, so the first pass cannot verify.  The forced
	// reload that follows finds the key the peer is signing with now.
	_, stalePEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: stalePEM, rotateTo: rotatedPEM, fromCache: true}

	signature, err := verifySignature(request, store.load)

	require.NoError(t, err)
	require.Equal(t, "https://remote.example/@alice", signature.ActorID())
	require.Equal(t, 2, store.loads, "one cached read, then one forced reload")
}

// TestVerifySignature_RotationRepairedOnSignedPost is the same repair on the path that actually
// matters -- a POST to an inbox, where the signature covers a body Digest.
func TestVerifySignature_RotationRepairedOnSignedPost(t *testing.T) {

	// Two passes means sigs.Verify reads the request body TWICE. It survives that because
	// re.ReadRequestBody puts a fresh reader back each time; if that ever stopped being true, the
	// second pass would fail its digest check and rotation would silently stop working for inboxes --
	// which is every delivery Emissary receives.
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	body := `{"type":"Create","actor":"https://remote.example/@alice"}`
	request := httptest.NewRequest(http.MethodPost, "https://local.example/@bob/pub/inbox", strings.NewReader(body))
	require.NoError(t, sigs.Sign(request, signatureKeyID, privateKey))

	// The cache holds the peer's previous key; the reload finds the one they signed with
	_, stalePEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: stalePEM, rotateTo: sigs.EncodePublicPEM(privateKey), fromCache: true}

	signature, err := verifySignature(request, store.load)

	require.NoError(t, err)
	require.Equal(t, "https://remote.example/@alice", signature.ActorID())
	require.Equal(t, 2, store.loads)
}

// TestVerifySignature_FreshKeyIsNotRefetched is the terminating case: a key that already came from
// the origin cannot be stale, so a failure against it is final.
func TestVerifySignature_FreshKeyIsNotRefetched(t *testing.T) {

	request, _ := newSignedRequest(t)

	// A key that does not match the signature, served as a FRESH (non-cached) answer
	_, wrongPEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: wrongPEM, fromCache: false}

	_, err := verifySignature(request, store.load)

	require.Error(t, err)
	require.Equal(t, 1, store.loads, "a fresh key must not trigger a reload")
}

// TestVerifySignature_UnchangedKeyIsNotReverified pins the second half of the refresh rule: the
// reload happens, but an unchanged key ends the attempt rather than paying for a second crypto check.
func TestVerifySignature_UnchangedKeyIsNotReverified(t *testing.T) {

	request, _ := newSignedRequest(t)

	// A cached key that does not match, and an origin that keeps returning that same key
	_, wrongPEM := newSignedRequest(t)
	store := &fakeKeyStore{publicKeyPEM: wrongPEM, fromCache: true}

	_, err := verifySignature(request, store.load)

	require.Error(t, err)
	require.Equal(t, 2, store.loads, "exactly one reload, and no third attempt")
}

// TestVerifySignature_NoKeyLookupBeforeParsing confirms that a request the verifier rejects on its own
// terms never reaches the key store at all.
func TestVerifySignature_NoKeyLookupBeforeParsing(t *testing.T) {

	// A garbage Signature header is refused during parsing, so no keyId is ever resolved -- this is
	// what keeps an unparseable request from costing an outbound fetch.
	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)
	request.Header.Set("Signature", "this-is-not-a-signature")

	store := &fakeKeyStore{publicKeyPEM: "irrelevant", fromCache: true}

	_, err := verifySignature(request, store.load)

	require.Error(t, err)
	require.Equal(t, 0, store.loads)
}

// TestVerifySignature_UnloadableKeyIsNotRetried covers the peer we cannot reach: the first pass already
// went to the network and failed, so a second attempt would only repeat it.
func TestVerifySignature_UnloadableKeyIsNotRetried(t *testing.T) {

	request, _ := newSignedRequest(t)
	store := &fakeKeyStore{err: derp.Internal("test", "Remote host unreachable")}

	_, err := verifySignature(request, store.load)

	require.Error(t, err)
	require.Equal(t, 1, store.loads)
}

// TestRotatedKeyFinder confirms that the second pass serves its key only to the keyID it was loaded
// for.  Serving it to any other keyID would authenticate a signature against a key its own keyId
// never named.
func TestRotatedKeyFinder(t *testing.T) {

	finder := rotatedKeyFinder(signatureKeyID, "PEM-ROTATED")

	publicKeyPEM, err := finder(signatureKeyID)
	require.NoError(t, err)
	require.Equal(t, "PEM-ROTATED", publicKeyPEM)

	_, err = finder("https://evil.example/@mallory#main-key")
	require.Error(t, err)
}
