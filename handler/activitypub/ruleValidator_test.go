package activitypub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/validator"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// fakeChecker is a RuleChecker that returns a canned disposition and records how it was called, so
// the tests can assert both the RESULT and the KEYS the validator derived from the request.
type fakeChecker struct {
	disposition model.RuleDisposition
	err         error
	called      bool
	gotKeys     []string
}

// DispositionForKeys implements the rule Checker interface, recording the keys it was asked about
func (c *fakeChecker) DispositionForKeys(_ data.Session, _ primitive.ObjectID, keys []string, _ int64) (model.RuleDisposition, error) {
	c.called = true
	c.gotKeys = keys
	return c.disposition, c.err
}

// document builds a minimal *streams.Document with the given actor and activity type.
func document(actorID string, activityType string) *streams.Document {
	doc := streams.NewDocument(mapof.Any{
		vocab.PropertyActor: actorID,
		vocab.PropertyType:  activityType,
	})
	return &doc
}

// request builds a POST inbox request. When signatureKeyID is non-empty it carries a parseable HTTP
// Signature header with that keyId; the signature bytes are irrelevant because Stage 1 only reads the
// keyId host and never verifies.
func request(signatureKeyID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "http://localhost/inbox", nil)
	if signatureKeyID != "" {
		req.Header.Set("Signature", `keyId="`+signatureKeyID+`",headers="(request-target) host date",signature="AAAA"`)
	}
	return req
}

// A blocked claimed actor is discarded pre-verification, and the actor's own keys reach the checker.
func TestRuleValidator_BlockedActor(t *testing.T) {

	checker := &fakeChecker{disposition: model.RuleDisposition{Action: model.RuleActionBlock}}
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	result := subject.Validate(request(""), document("https://evil.example/@spammer", vocab.ActivityTypeCreate))

	require.Equal(t, validator.ResultInvalid, result)
	require.True(t, checker.called)
	require.Contains(t, checker.gotKeys, "ACTOR:https://evil.example/@spammer")
}

// The signature keyId host is folded in as DOMAIN keys, so a blocked relay is caught even when the
// claimed actor is clean -- and the key is never fetched.
func TestRuleValidator_KeyIDHostFoldedIn(t *testing.T) {

	checker := &fakeChecker{} // returns no disposition; we only assert the derived keys
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	result := subject.Validate(request("https://relay.evil.example/actor#main-key"), document("https://good.example/@friend", vocab.ActivityTypeCreate))

	require.Equal(t, validator.ResultUnknown, result)
	require.Contains(t, checker.gotKeys, "DOMAIN:relay.evil.example")
	require.Contains(t, checker.gotKeys, "DOMAIN:evil.example")
}

// Exception-set types (Follow/Delete/Undo/Move) are verified first, never fast-discarded -- the
// checker is not even consulted.
func TestRuleValidator_ExceptionTypesDeferred(t *testing.T) {

	for _, activityType := range []string{vocab.ActivityTypeFollow, vocab.ActivityTypeDelete, vocab.ActivityTypeUndo, vocab.ActivityTypeMove} {

		checker := &fakeChecker{disposition: model.RuleDisposition{Action: model.RuleActionBlock}}
		subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

		result := subject.Validate(request(""), document("https://evil.example/@spammer", activityType))

		require.Equal(t, validator.ResultUnknown, result, activityType)
		require.False(t, checker.called, activityType)
	}
}

// Inline non-public MLS is never fast-discarded (4B), even from a blocked actor -- and since no
// discard is possible, the checker is not consulted at all.
func TestRuleValidator_MLSCreateNeverDiscarded(t *testing.T) {

	checker := &fakeChecker{disposition: model.RuleDisposition{Action: model.RuleActionBlock}}
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	doc := streams.NewDocument(mlsCreate())
	result := subject.Validate(request(""), &doc)

	require.Equal(t, validator.ResultUnknown, result)
	require.False(t, checker.called)
}

// A clean actor defers to signature verification.
func TestRuleValidator_CleanActor(t *testing.T) {

	checker := &fakeChecker{}
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	result := subject.Validate(request(""), document("https://good.example/@friend", vocab.ActivityTypeCreate))

	require.Equal(t, validator.ResultUnknown, result)
}

// A checker error fails OPEN to Stage 2 (D17): the optimization layer must not break all federation.
func TestRuleValidator_ErrorFailsOpen(t *testing.T) {

	checker := &fakeChecker{err: derp.Internal("test", "database is on fire")}
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	result := subject.Validate(request(""), document("https://evil.example/@spammer", vocab.ActivityTypeCreate))

	require.Equal(t, validator.ResultUnknown, result)
}

// MUTE never gates the wire (D5): a muted actor is not discarded.
func TestRuleValidator_MuteDoesNotGate(t *testing.T) {

	checker := &fakeChecker{disposition: model.RuleDisposition{Action: model.RuleActionMute}}
	subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

	result := subject.Validate(request(""), document("https://noisy.example/@chatty", vocab.ActivityTypeCreate))

	require.Equal(t, validator.ResultUnknown, result)
}

// Stage 1 must NEVER return ResultValid, for any input -- that would short-circuit the chain and
// bypass signature verification. Sweep every disposition x a representative type set.
func TestRuleValidator_NeverReturnsValid(t *testing.T) {

	dispositions := []model.RuleDisposition{
		{},
		{Action: model.RuleActionBlock},
		{Action: model.RuleActionMute},
	}
	types := []string{vocab.ActivityTypeCreate, vocab.ActivityTypeFollow, vocab.ActivityTypeAnnounce, vocab.ActivityTypeLike}

	for _, disposition := range dispositions {
		for _, activityType := range types {

			checker := &fakeChecker{disposition: disposition}
			subject := NewRuleValidator(checker, nil, primitive.NilObjectID)

			result := subject.Validate(request(""), document("https://any.example/@actor", activityType))

			require.NotEqual(t, validator.ResultValid, result)
		}
	}
}
