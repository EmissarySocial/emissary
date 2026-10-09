package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/sigs"
	"github.com/stretchr/testify/require"
)

/******************************************
 * Signature Resolution Tests
 *
 * These pin the three-case rule that resolveSignature keeps apart: no
 * signature (Anonymous), a valid one (an Actor), and one that FAILS to
 * verify (a refusal). Collapsing the third into the first is BUG-20.
 *
 * resolveSignature takes its verifier as an argument, so no case here
 * needs a Factory, a database, or real crypto -- the routing between
 * the three is the whole subject.
 ******************************************/

// testSignatureKeyID is the keyId carried by the signatures in these tests.
const testSignatureKeyID = "https://remote.example/@alice#main-key"

// signedRequest returns a request that carries a (syntactically present) Signature header.
func signedRequest(host string) *http.Request {

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub", nil)
	request.Host = host
	request.Header.Set("Signature", `keyId="`+testSignatureKeyID+`",signature="AAAA"`)

	return request
}

// unsignedRequest returns a request that carries no Signature header at all.
func unsignedRequest(host string) *http.Request {

	request := httptest.NewRequest(http.MethodGet, "https://local.example/@bob/pub", nil)
	request.Host = host

	return request
}

// verifierSucceeds returns a verifier that accepts every signature, naming testSignatureKeyID.
func verifierSucceeds() func(*http.Request) (sigs.Signature, error) {

	return func(*http.Request) (sigs.Signature, error) {
		signature := sigs.NewSignature()
		signature.KeyID = testSignatureKeyID
		return signature, nil
	}
}

// verifierFails returns a verifier that rejects every signature, carrying a detail string that
// must never reach the caller.
func verifierFails() func(*http.Request) (sigs.Signature, error) {

	return func(*http.Request) (sigs.Signature, error) {
		return sigs.Signature{}, derp.Internal("test.verifier", "SECRET-INTERNAL-DETAIL")
	}
}

// verifierUnused returns a verifier that fails the test if it is ever called.
func verifierUnused(t *testing.T) func(*http.Request) (sigs.Signature, error) {

	t.Helper()

	return func(*http.Request) (sigs.Signature, error) {
		t.Fatal("verifier must not be called for an unsigned request")
		return sigs.Signature{}, nil
	}
}

// TestResolveSignature_Unsigned pins the case that must NOT regress: a request with no Signature
// header is Anonymous, not refused, and never reaches the verifier.
func TestResolveSignature_Unsigned(t *testing.T) {

	t.Parallel()

	signature, err := resolveSignature(unsignedRequest("example.com"), verifierUnused(t))

	require.NoError(t, err)
	require.Empty(t, signature.KeyID)
}

// TestResolveSignature_Valid confirms that a verified signature is passed straight through.
func TestResolveSignature_Valid(t *testing.T) {

	t.Parallel()

	signature, err := resolveSignature(signedRequest("example.com"), verifierSucceeds())

	require.NoError(t, err)
	require.Equal(t, testSignatureKeyID, signature.KeyID)
}

// TestResolveSignature_Invalid is the heart of BUG-20: a signature that fails to verify refuses
// the request instead of downgrading it to Anonymous.
func TestResolveSignature_Invalid(t *testing.T) {

	t.Parallel()

	signature, err := resolveSignature(signedRequest("example.com"), verifierFails())

	require.Error(t, err)
	require.Equal(t, http.StatusUnauthorized, derp.ErrorCode(err))
	require.Empty(t, signature.KeyID)
}

// TestResolveSignature_RefusalLeaksNothing guarantees that the refusal handed to an
// unauthenticated caller carries no verifier internals.
func TestResolveSignature_RefusalLeaksNothing(t *testing.T) {

	t.Parallel()

	_, err := resolveSignature(signedRequest("example.com"), verifierFails())

	require.Error(t, err)
	require.Equal(t, "Invalid HTTP Signature", derp.Message(err))
	require.NotContains(t, derp.Message(err), "SECRET-INTERNAL-DETAIL")
}

// TestResolveSignature_RefusalStaysOutOfDerp pins the cross-file invariant that keeps refusals
// out of the production error log: the refusal must be an Unauthorized error.
func TestResolveSignature_RefusalStaysOutOfDerp(t *testing.T) {

	t.Parallel()

	_, err := resolveSignature(signedRequest("example.com"), verifierFails())

	require.Error(t, err)
	require.True(t, derp.IsUnauthorized(err), "refusal must be Unauthorized so errorHandler skips derp.Report")
}

// TestResolveSignature_LocalRefusalIsFixed confirms that a local domain gets the same fixed
// refusal as any other, with no verifier internals in it.
func TestResolveSignature_LocalRefusalIsFixed(t *testing.T) {

	t.Parallel()

	_, err := resolveSignature(signedRequest("localhost"), verifierFails())

	require.Error(t, err)
	require.True(t, derp.IsUnauthorized(err))
	require.Equal(t, "Invalid HTTP Signature", derp.Message(err))
}

// TestParseHTTPSignature_NilRequest pins the guard at the top of ParseHTTPSignature: a nil
// request grants anonymous permissions and is not an error.
func TestParseHTTPSignature_NilRequest(t *testing.T) {

	t.Parallel()

	// A zero-value service is safe here: neither path below reaches a collaborator.
	permissionService := NewPermission()

	permissions, err := permissionService.ParseHTTPSignature(nil, nil)

	require.NoError(t, err)
	require.Equal(t, model.NewAnonymousPermissions(), permissions)
}

// TestParseHTTPSignature_Unsigned pins the whole anonymous path through the service method: an
// unsigned request never reaches the verifier, and returns before any Identity is loaded.
func TestParseHTTPSignature_Unsigned(t *testing.T) {

	t.Parallel()

	permissionService := NewPermission()

	permissions, err := permissionService.ParseHTTPSignature(nil, unsignedRequest("example.com"))

	require.NoError(t, err)
	require.Equal(t, model.NewAnonymousPermissions(), permissions)
}
