package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestRule_ActivityType_NoFlag is the regression test for R15: a LABEL rule must never map to a
// "Flag". Everywhere else in the Fediverse a Flag is a PRIVATE moderation report to a server's
// moderators, so publishing personal labels as Flags broadcasts what conformant receivers read as
// an abuse report against the labelled Actor. BLOCK and MUTE keep their mappings.
func TestRule_ActivityType_NoFlag(t *testing.T) {

	service := Rule{}

	rule := model.NewRule()

	rule.Action = model.RuleActionBlock
	require.Equal(t, vocab.ActivityTypeBlock, service.ActivityType(rule))

	rule.Action = model.RuleActionMute
	require.Equal(t, vocab.ActivityTypeIgnore, service.ActivityType(rule))

	// The whole point of Phase 1: LABEL maps to nothing at all.
	rule.Action = model.RuleActionLabel
	require.Equal(t, "", service.ActivityType(rule))
	require.NotEqual(t, vocab.ActivityTypeFlag, service.ActivityType(rule))

	// An unrecognized Action federates as nothing, rather than defaulting to some type.
	rule.Action = "SOMETHING-NEW"
	require.Equal(t, "", service.ActivityType(rule))
}

// TestRule_shouldPublish covers the gate that decides whether a Rule federates, for the only Rules
// eligible to: Domain-owned ones. A Domain Rule has no owning User, which is what OriginAdmin tests.
func TestRule_shouldPublish(t *testing.T) {

	service := Rule{}

	rule := model.NewRule()
	rule.UserID = primitive.NilObjectID
	require.True(t, rule.OriginAdmin(), "guard: this fixture must be a Domain Rule")

	// A private rule never publishes, whatever its Action.
	rule.IsPublic = false

	for _, action := range []string{model.RuleActionBlock, model.RuleActionMute, model.RuleActionLabel} {
		rule.Action = action
		require.False(t, service.shouldPublish(rule))
	}

	// A public rule publishes -- unless it is a LABEL.
	rule.IsPublic = true

	rule.Action = model.RuleActionBlock
	require.True(t, service.shouldPublish(rule))

	rule.Action = model.RuleActionMute
	require.True(t, service.shouldPublish(rule))

	rule.Action = model.RuleActionLabel
	require.False(t, service.shouldPublish(rule))
}

// TestRule_shouldPublish_UsersNeverPublish is the regression test for D9: federating moderation
// policy is a Domain act. A User's Rules are private no matter what the record says -- which is why
// this asserts against the combination that WOULD have published before D9 (public, non-LABEL).
func TestRule_shouldPublish_UsersNeverPublish(t *testing.T) {

	service := Rule{}

	rule := model.NewRule()
	rule.UserID = primitive.NewObjectID()
	rule.IsPublic = true

	require.False(t, rule.OriginAdmin(), "guard: this fixture must be a User Rule")

	for _, action := range []string{model.RuleActionBlock, model.RuleActionMute, model.RuleActionLabel} {
		rule.Action = action
		require.False(t, service.shouldPublish(rule), "a User's Rule must never federate: "+action)
	}
}

// TestRule_shouldPublish_DoesNotMutate pins shouldPublish as a pure predicate. It reads IsPublic to
// decide what federates; it never writes it. Deciding what a Rule MEANS is a caller's job, and a gate
// that quietly rewrote the record would make PublishDate and IsPublic disagree about the same Rule.
func TestRule_shouldPublish_DoesNotMutate(t *testing.T) {

	service := Rule{}

	rule := model.NewRule()
	rule.Action = model.RuleActionLabel
	rule.IsPublic = true

	require.False(t, service.shouldPublish(rule))
	require.True(t, rule.IsPublic, "shouldPublish must not rewrite the Rule it is asked about")
}

// TestRule_JSONLD_LabelHasNoType documents the reason the LABEL type must be made UNREACHABLE rather
// than merely unmapped: JSONLD writes `type` unconditionally, so a LABEL rule that reached this far
// would serialize an explicit empty type -- invalid ActivityStreams, and worse than the original bug
// because an empty string defeats a receiver's presence check. shouldPublish is what keeps this
// document off the wire.
func TestRule_JSONLD_LabelHasNoType(t *testing.T) {

	service := Rule{}

	rule := model.NewRule()
	rule.Action = model.RuleActionLabel
	rule.Type = model.RuleTypeActor
	rule.Trigger = "https://example.com/@spammer"

	require.Equal(t, "", service.JSONLD(rule)[vocab.PropertyType])

	// A BLOCK, by contrast, is well-formed and safe to publish.
	rule.Action = model.RuleActionBlock
	require.Equal(t, vocab.ActivityTypeBlock, service.JSONLD(rule)[vocab.PropertyType])
}
