package handler

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/stretchr/testify/require"
)

// testKeyID is the keyId used by the signing fixtures below. The fragment is the conventional
// actor-URL-plus-hash shape, so ActorID() should cut it back to the actor.
const testKeyID = "https://remote.example/@alice#main-key"

// testActorID is the Actor that testKeyID resolves to.
const testActorID = "https://remote.example/@alice"

// signedRequest returns a GET request signed by a freshly generated key, plus a PublicKeyFinder
// that resolves testKeyID to the matching public key.
func signedRequest(t *testing.T) (*http.Request, sigs.PublicKeyFinder) {

	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)
	require.NoError(t, sigs.Sign(request, testKeyID, privateKey))

	publicKeyPEM := sigs.EncodePublicPEM(privateKey)

	finder := func(keyID string) (string, error) {
		if keyID != testKeyID {
			return "", derp.NotFound("test.finder", "Unknown key", keyID)
		}
		return publicKeyPEM, nil
	}

	return request, finder
}

// unusedFinder is a PublicKeyFinder that fails the test if it is ever called. It proves that the
// unsigned path never reaches out for a key.
func unusedFinder(t *testing.T) sigs.PublicKeyFinder {

	t.Helper()

	return func(keyID string) (string, error) {
		t.Fatalf("PublicKeyFinder must not be called; got %q", keyID)
		return "", nil
	}
}

// TestResolveSignedActor_Unsigned pins the case that must NOT regress: a request with no Signature
// header is Anonymous, not refused. BUG-20 narrowed only the third case.
func TestResolveSignedActor_Unsigned(t *testing.T) {

	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)

	actorID, err := resolveSignedActor(request, unusedFinder(t))

	require.NoError(t, err)
	require.Empty(t, actorID)
}

// TestResolveSignedActor_Valid confirms that a verified signature yields its Actor, and that the
// keyId fragment is cut back to the Actor URL.
func TestResolveSignedActor_Valid(t *testing.T) {

	t.Parallel()

	request, finder := signedRequest(t)

	actorID, err := resolveSignedActor(request, finder)

	require.NoError(t, err)
	require.Equal(t, testActorID, actorID)
}

// TestResolveSignedActor_Corrupted is the heart of BUG-20: a Signature header that cannot be parsed
// must produce a 401, NOT an anonymous request.
func TestResolveSignedActor_Corrupted(t *testing.T) {

	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)
	request.Header.Set("Signature", "this-is-not-a-signature")

	actorID, err := resolveSignedActor(request, unusedFinder(t))

	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))
	require.Empty(t, actorID)
}

// TestResolveSignedActor_Tampered covers the forged case: the signature parses and names a key we
// can resolve, but it does not verify against the request.
func TestResolveSignedActor_Tampered(t *testing.T) {

	t.Parallel()

	request, finder := signedRequest(t)

	// Alter a signed header AFTER signing, so the digest no longer matches
	request.Header.Set("Date", "Tue, 01 Jan 2030 00:00:00 GMT")

	actorID, err := resolveSignedActor(request, finder)

	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))
	require.Empty(t, actorID)
}

// TestResolveSignedActor_KeyUnresolvable pins the availability trade recorded in BUG-20: when the
// peer's key cannot be fetched, the request is REFUSED rather than downgraded to anonymous.
func TestResolveSignedActor_KeyUnresolvable(t *testing.T) {

	t.Parallel()

	request, _ := signedRequest(t)

	finder := func(keyID string) (string, error) {
		return "", derp.Internal("test.finder", "Remote host unreachable")
	}

	actorID, err := resolveSignedActor(request, finder)

	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))
	require.Empty(t, actorID)
}

// TestResolveSignedActor_RefusalLeaksNothing guarantees that the refusal handed to an unauthenticated
// caller carries no verifier internals. errorHandler writes derp.Message() into the response body, so
// anything the verifier said about WHY it failed would tell a prober which attempt got closest.
func TestResolveSignedActor_RefusalLeaksNothing(t *testing.T) {

	t.Parallel()

	request, _ := signedRequest(t)

	finder := func(keyID string) (string, error) {
		return "", derp.Internal("test.finder", "SECRET-INTERNAL-DETAIL")
	}

	_, err := resolveSignedActor(request, finder)

	require.Error(t, err)
	require.Equal(t, "Invalid HTTP Signature", derp.Message(err))
	require.NotContains(t, derp.Message(err), "SECRET-INTERNAL-DETAIL")
}

// TestResolveSignedActor_RefusalStaysOutOfDerp pins the cross-file invariant that BUG-20 depends on:
// the refusal must be an Unauthorized error, because server.errorHandler answers 401s and returns
// BEFORE derp.Report. A refactor that changed this code to any other status would silently start
// filing every misconfigured peer into the production error log.
func TestResolveSignedActor_RefusalStaysOutOfDerp(t *testing.T) {

	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub/objects/123", nil)
	request.Header.Set("Signature", "this-is-not-a-signature")

	_, err := resolveSignedActor(request, unusedFinder(t))

	require.Error(t, err)
	require.True(t, derp.IsUnauthorized(err), "refusal must be Unauthorized so errorHandler skips derp.Report")
}
