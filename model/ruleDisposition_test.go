package model

import (
	"reflect"
	"strings"
	"testing"

	"github.com/benpate/hannibal/streams"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// testNow is the fixed timestamp that the rule-evaluation tests below are anchored to
const testNow = int64(1_000_000)

// actorDocument builds a minimal document attributed to the given actor URI.
func actorDocument(actorURI string) streams.Document {
	return streams.NewDocument(mapof.Any{vocab.PropertyActor: actorURI})
}

// actorRule builds a RuleSummary that matches the given actor URI.
func actorRule(userID primitive.ObjectID, action string, actorURI string) RuleSummary {
	return RuleSummary{
		RuleID:   primitive.NewObjectID(),
		UserID:   userID,
		Type:     RuleTypeActor,
		Action:   action,
		Trigger:  actorURI,
		MatchKey: RuleMatchKey(RuleTypeActor, actorURI),
	}
}

// TestNewRuleDispositionForKeys confirms the wire-gate entry point evaluates rules against an
// explicit key set (no document): a DOMAIN block on the delivering host wins even when the caller
// passes only actor/domain keys, and an unrelated key set produces no disposition.
func TestNewRuleDispositionForKeys(t *testing.T) {
	user := primitive.NewObjectID()

	domainBlock := RuleSummary{
		RuleID:   primitive.NewObjectID(),
		UserID:   user,
		Type:     RuleTypeDomain,
		Action:   RuleActionBlock,
		Trigger:  "evil.com",
		MatchKey: RuleMatchKey(RuleTypeDomain, "evil.com"),
	}

	// The delivering actor's keys include the blocked domain suffix -> blocked.
	blocked := NewRuleDispositionForKeys(ActorMatchKeys("https://sub.evil.com/@spammer"), []RuleSummary{domainBlock}, testNow)
	require.True(t, blocked.IsBlocked())
	require.Equal(t, domainBlock.RuleID, blocked.RuleID)

	// A different origin's keys never reach the rule.
	clean := NewRuleDispositionForKeys(ActorMatchKeys("https://good.example/@friend"), []RuleSummary{domainBlock}, testNow)
	require.False(t, clean.IsFiltered())
}

// TestEvaluate_NoRules verifies that a document matched by no rules is left clean
func TestEvaluate_NoRules(t *testing.T) {
	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), nil, testNow)
	require.Equal(t, RuleDispositionNone, disposition.Action)
	require.False(t, disposition.IsFiltered())
}

// TestEvaluate_Block verifies that a matching BLOCK rule blocks the document
func TestEvaluate_Block(t *testing.T) {
	rule := actorRule(primitive.NewObjectID(), RuleActionBlock, "https://example.com/@bob")
	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{rule}, testNow)

	require.True(t, disposition.IsBlocked())
	require.Equal(t, rule.RuleID, disposition.RuleID)
	require.Equal(t, RuleOriginUser, disposition.Tier)
}

// TestEvaluate_MaxSeverityWins verifies that BLOCK beats MUTE, whatever order the rules arrive in
func TestEvaluate_MaxSeverityWins(t *testing.T) {
	user := primitive.NewObjectID()
	mute := actorRule(user, RuleActionMute, "https://example.com/@bob")
	block := actorRule(user, RuleActionBlock, "https://example.com/@bob")

	// Regardless of order, BLOCK (higher severity) wins.
	require.True(t, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{mute, block}, testNow).IsBlocked())
	require.True(t, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{block, mute}, testNow).IsBlocked())
}

// TestEvaluate_MuteOnly verifies that a matching MUTE rule mutes without blocking
func TestEvaluate_MuteOnly(t *testing.T) {
	rule := actorRule(primitive.NewObjectID(), RuleActionMute, "https://example.com/@bob")
	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{rule}, testNow)

	require.True(t, disposition.IsMuted())
	require.True(t, disposition.IsFiltered())
	require.False(t, disposition.IsBlocked())
}

// TestEvaluate_LabelDoesNotFilter verifies that a LABEL rule annotates a document without filtering it
func TestEvaluate_LabelDoesNotFilter(t *testing.T) {
	rule := actorRule(primitive.NewObjectID(), RuleActionLabel, "https://example.com/@bob")
	rule.Label = "State media"
	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{rule}, testNow)

	require.Equal(t, RuleDispositionNone, disposition.Action)
	require.False(t, disposition.IsFiltered())
	require.True(t, disposition.HasLabels())
	require.Equal(t, "State media", disposition.Labels[0].Label)
}

// TestEvaluate_LabelsCollectedUnderBlock verifies that labels are still collected when a BLOCK also matches
func TestEvaluate_LabelsCollectedUnderBlock(t *testing.T) {
	user := primitive.NewObjectID()
	block := actorRule(user, RuleActionBlock, "https://example.com/@bob")
	label := actorRule(user, RuleActionLabel, "https://example.com/@bob")
	label.Label = "Explains the block"

	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{block, label}, testNow)

	require.True(t, disposition.IsBlocked())
	require.True(t, disposition.HasLabels(), "labels are collected even when the final action is BLOCK")
}

// TestEvaluate_ExpiredRuleSkipped verifies that an expired rule is ignored, while a live one still applies
func TestEvaluate_ExpiredRuleSkipped(t *testing.T) {
	expired := actorRule(primitive.NewObjectID(), RuleActionBlock, "https://example.com/@bob")
	expired.ExpireDate = testNow - 1 // already passed

	require.False(t, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{expired}, testNow).IsFiltered())

	// A not-yet-expired rule still applies.
	future := actorRule(primitive.NewObjectID(), RuleActionBlock, "https://example.com/@bob")
	future.ExpireDate = testNow + 1
	require.True(t, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{future}, testNow).IsBlocked())
}

// TestEvaluate_TieBreaksToUser pins D14: when an ADMIN and a USER rule match at the same severity,
// the USER rule is named the winner (attribution only). Order must not matter.
func TestEvaluate_TieBreaksToUser(t *testing.T) {
	admin := actorRule(primitive.NilObjectID, RuleActionBlock, "https://example.com/@bob")
	user := actorRule(primitive.NewObjectID(), RuleActionBlock, "https://example.com/@bob")

	require.Equal(t, RuleOriginUser, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{admin, user}, testNow).Tier)
	require.Equal(t, RuleOriginUser, NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{user, admin}, testNow).Tier)
}

// TestEvaluate_AdminFloorHolds confirms an ADMIN block still blocks when the only user-tier rule is a
// weaker (or absent) action -- the "admin is a floor" theorem.
func TestEvaluate_AdminFloorHolds(t *testing.T) {
	adminBlock := actorRule(primitive.NilObjectID, RuleActionBlock, "https://example.com/@bob")
	userLabel := actorRule(primitive.NewObjectID(), RuleActionLabel, "https://example.com/@bob")

	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{userLabel, adminBlock}, testNow)
	require.True(t, disposition.IsBlocked())
	require.Equal(t, RuleOriginAdmin, disposition.Tier)
}

// TestEvaluate_NonMatchingRuleIgnored confirms a rule whose key is not among the document's keys is
// never applied -- the query pre-filters, but the engine re-checks so a broad candidate set is safe.
func TestEvaluate_NonMatchingRuleIgnored(t *testing.T) {
	other := actorRule(primitive.NewObjectID(), RuleActionBlock, "https://elsewhere.example/@carol")
	disposition := NewRuleDisposition(actorDocument("https://example.com/@bob"), []RuleSummary{other}, testNow)
	require.False(t, disposition.IsFiltered())
}

// TestRuleSummaryFields_PinnedToStruct is the guard against the silent all-ADMIN-tier failure: every
// bson field on RuleSummary MUST be in the projection, or a dropped field (esp. userId) reads as its
// zero value with no error.
func TestRuleSummaryFields_PinnedToStruct(t *testing.T) {

	fields := RuleSummaryFields()
	summaryType := reflect.TypeOf(RuleSummary{})

	for i := range summaryType.NumField() {

		bsonName, _, _ := strings.Cut(summaryType.Field(i).Tag.Get("bson"), ",")

		if (bsonName == "") || (bsonName == "-") {
			continue
		}

		require.Contains(t, fields, bsonName, "RuleSummaryFields() must project struct field %q (bson %q)", summaryType.Field(i).Name, bsonName)
	}
}
