package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// fakeActorLoader is a stub actorLoader that records whether GetActor was called and returns either a
// document whose id is `result`, or `err`. It lets resolveMatchKeyTrigger be tested without the network.
type fakeActorLoader struct {
	result string
	err    error
	called bool
}

// GetActor implements the actor-loader interface, returning this stub's canned document or error
func (loader *fakeActorLoader) GetActor(address string) (streams.Document, error) {
	loader.called = true

	if loader.err != nil {
		return streams.NilDocument(), loader.err
	}

	return streams.NewDocument(mapof.Any{vocab.PropertyID: loader.result}), nil
}

// TestResolveMatchKeyTrigger_HandleResolved proves the core fix: a hand-typed webfinger handle is
// resolved to the canonical actor URL for the MATCH KEY, while the Trigger the user typed is left
// untouched (friendly display value preserved).
func TestResolveMatchKeyTrigger_HandleResolved(t *testing.T) {

	const canonical = "https://alice.local/@6a536bfc"

	loader := &fakeActorLoader{result: canonical}
	service := &Rule{activityStreamService: loader}

	rule := model.NewRule()
	rule.Type = model.RuleTypeActor
	rule.Trigger = "@david@alice.local"

	keyTrigger, err := service.resolveMatchKeyTrigger(&rule)

	require.Nil(t, err)
	require.True(t, loader.called)
	require.Equal(t, canonical, keyTrigger)              // the key is derived from the canonical id
	require.Equal(t, "@david@alice.local", rule.Trigger) // the user's input is NOT rewritten
}

// TestResolveMatchKeyTrigger_MatchesInboundActivity is the regression proof: BEFORE resolution the
// handle keys to something no activity produces (inert -- the reported bug); AFTER resolution the
// derived key IS among the inbound activity's own keys, all while Trigger stays the friendly handle.
func TestResolveMatchKeyTrigger_MatchesInboundActivity(t *testing.T) {

	const canonical = "https://alice.local/@6a536bfc"

	// The keys an inbound activity from this actor produces.
	inbound := model.DocumentMatchKeys(streams.NewDocument(mapof.Any{vocab.PropertyActor: canonical}))

	// Keying on the raw handle produces a value NOT among them -- the rule would never match.
	rawKey := model.RuleMatchKey(model.RuleTypeActor, "@david@alice.local")
	require.NotContains(t, inbound, rawKey)

	// Keying on the resolved value produces a key that IS among the inbound activity's keys.
	loader := &fakeActorLoader{result: canonical}
	service := &Rule{activityStreamService: loader}

	rule := model.NewRule()
	rule.Type = model.RuleTypeActor
	rule.Trigger = "@david@alice.local"

	keyTrigger, err := service.resolveMatchKeyTrigger(&rule)
	require.Nil(t, err)

	resolvedKey := model.RuleMatchKey(model.RuleTypeActor, keyTrigger)
	require.Contains(t, inbound, resolvedKey)
	require.Equal(t, "@david@alice.local", rule.Trigger) // still the friendly value
}

// TestResolveMatchKeyTrigger_UnresolvableRefused proves an address that will not resolve is REFUSED
// with a Validation (422) error -- never saved inert.
func TestResolveMatchKeyTrigger_UnresolvableRefused(t *testing.T) {

	loader := &fakeActorLoader{err: derp.NotFound("test", "no such actor")}
	service := &Rule{activityStreamService: loader}

	rule := model.NewRule()
	rule.Type = model.RuleTypeActor
	rule.Trigger = "@ghost@nowhere.invalid"

	keyTrigger, err := service.resolveMatchKeyTrigger(&rule)

	require.NotNil(t, err)
	require.Equal(t, 422, derp.ErrorCode(err))
	require.Equal(t, "", keyTrigger)
}

// TestResolveMatchKeyTrigger_NonActorNoop proves DOMAIN and TAG rules never touch the network: the
// loader is not called and the Trigger is returned unchanged. (A loader that would error confirms it.)
func TestResolveMatchKeyTrigger_NonActorNoop(t *testing.T) {

	for _, ruleType := range []string{model.RuleTypeDomain, model.RuleTypeTag} {

		loader := &fakeActorLoader{err: derp.Internal("test", "must not be called")}
		service := &Rule{activityStreamService: loader}

		rule := model.NewRule()
		rule.Type = ruleType
		rule.Trigger = "example.com"

		keyTrigger, err := service.resolveMatchKeyTrigger(&rule)

		require.Nil(t, err)
		require.False(t, loader.called)
		require.Equal(t, "example.com", keyTrigger)
		require.Equal(t, "example.com", rule.Trigger)
	}
}

// TestResolveMatchKeyTrigger_EmptyTriggerNoop proves an empty ACTOR trigger is left to schema
// validation rather than sent to the network resolver.
func TestResolveMatchKeyTrigger_EmptyTriggerNoop(t *testing.T) {

	loader := &fakeActorLoader{err: derp.Internal("test", "must not be called")}
	service := &Rule{activityStreamService: loader}

	rule := model.NewRule()
	rule.Type = model.RuleTypeActor
	rule.Trigger = ""

	keyTrigger, err := service.resolveMatchKeyTrigger(&rule)

	require.Nil(t, err)
	require.False(t, loader.called)
	require.Equal(t, "", keyTrigger)
}
