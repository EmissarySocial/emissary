package activitypub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/router"
	"github.com/benpate/hannibal/validator"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeKeyFinder is a PublicKeyProvider that records whether its finders were called.
type fakeKeyFinder struct {
	called    bool   // A call stands in for the outbound key fetch, so "never called" means no fetch happened
	refreshed bool   // TRUE if the REFRESH finder ran, which only happens after a signature fails
	gotKeyID  string // The keyID that PublicKeyFinder was asked to resolve
}

// PublicKeyFinder records the keyID it was asked for, then fails -- no test here needs a key that resolves.
func (f *fakeKeyFinder) PublicKeyFinder(keyID string) (string, error) {
	f.called = true
	f.gotKeyID = keyID
	return "", derp.NotFound("test", "no key for this test")
}

// RefreshPublicKey records that the refresh path ran.
func (f *fakeKeyFinder) RefreshPublicKey(keyID string) (string, error) {
	f.refreshed = true
	return "", derp.NotFound("test", "no key for this test")
}

// signedRequest builds a POST inbox request carrying the given JSON body and a parseable HTTP
// Signature that names the given keyID.
func signedRequest(keyID string, body string) *http.Request {

	// The signature bytes are garbage: these tests assert WHICH key finder runs (and whether it runs
	// at all), never that a signature verifies.

	result := httptest.NewRequest(http.MethodPost, "http://localhost/inbox", strings.NewReader(body))

	// The header list must cover the verifier's required fields, or sigs.Verify rejects the request
	// before it ever consults the key finder -- which would make these tests pass for the wrong reason.
	result.Header.Set("Signature", `keyId="`+keyID+`",headers="(request-target) host date digest",signature="AAAA"`)

	return result
}

// The key finder handed to the funnel is the one signature verification actually uses.
func TestReceiveRequest_UsesProvidedKeyFinder(t *testing.T) {

	// The core assertion of BUG-19: hannibal silently falls back to its own defaultKeyFinder when the
	// finder is nil, fetching keys with a bare HTTP GET outside Emissary's client stack.
	keyFinder := &fakeKeyFinder{}
	checker := &fakeChecker{} // no disposition: Stage 1 defers, so the chain reaches HTTPSig

	_, err := ReceiveRequest(
		signedRequest("https://good.example/@friend#main-key", `{"actor":"https://good.example/@friend","type":"Create"}`),
		nil,
		keyFinder,
		checker,
		nil,
		primitive.NilObjectID,
	)

	// Verification fails (the signature is garbage), but the finder ran -- which is the point.
	require.Error(t, err)
	require.True(t, keyFinder.called)
	require.Equal(t, "https://good.example/@friend#main-key", keyFinder.gotKeyID)

	// A finder that could not produce a key at all leaves the refresh nothing to improve on, so it
	// must not run.  Only a signature that FAILED against a real key is worth a second fetch.
	require.False(t, keyFinder.refreshed)
}

// A keyId pointing at a rules-blocked domain is discarded at Stage 1, and the key is NEVER fetched.
func TestReceiveRequest_BlockedKeyIDIsNotFetched(t *testing.T) {

	// Asserting the fetch did not happen (not merely that the request was rejected) is what closes the
	// amplification vector: a rejection AFTER the fetch still lets a blocked peer drive traffic.
	keyFinder := &fakeKeyFinder{}
	checker := &fakeChecker{disposition: model.RuleDisposition{Action: model.RuleActionBlock}}

	_, err := ReceiveRequest(
		signedRequest("https://relay.evil.example/actor#main-key", `{"actor":"https://good.example/@friend","type":"Create"}`),
		nil,
		keyFinder,
		checker,
		nil,
		primitive.NilObjectID,
	)

	require.Error(t, err)
	require.False(t, keyFinder.called)
	require.False(t, keyFinder.refreshed, "the refresh is a second fetch, and must be gated the same way")
	require.Contains(t, checker.gotKeys, "DOMAIN:relay.evil.example")
}

// The canonical chain is three validators deep, with Stage 1 ahead of signature verification.
func TestInboxValidators_ChainShape(t *testing.T) {

	// All four inbox families (user, stream, domain, search) receive through the one funnel, so the two
	// tests above hold for each of them by construction. This one pins the chain the funnel installs.
	keyFinder := &fakeKeyFinder{}
	config := router.NewReceiveConfig(InboxValidators(keyFinder, &fakeChecker{}, nil, primitive.NilObjectID))

	require.Len(t, config.Validators, 3)
	require.IsType(t, RuleValidator{}, config.Validators[0])
	require.IsType(t, validator.NewHTTPSig(nil), config.Validators[1])
	require.IsType(t, validator.NewDeletedObject(), config.Validators[2])

	// Stage 1 must come BEFORE signature verification, or a blocked origin's key is fetched anyway.
	require.Equal(t, 0, indexOfRuleValidator(config.Validators))
}

// indexOfRuleValidator returns the position of the Stage-1 RuleValidator in the chain, or -1.
func indexOfRuleValidator(validators []router.Validator) int {

	for index, item := range validators {
		if _, ok := item.(RuleValidator); ok {
			return index
		}
	}

	return -1
}
