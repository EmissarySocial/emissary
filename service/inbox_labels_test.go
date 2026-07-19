package service

import (
	"testing"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestInbox_LabeledChunkJSON pins the serve-time composition: served = persisted ∪ current, with
// one pre-fetched rule set evaluated per item and forged reserved keys always scrubbed.
func TestInbox_LabeledChunkJSON(t *testing.T) {

	now := int64(1_700_000_000)
	userRule := primitive.NewObjectID()

	// The viewer's CURRENT rules: one MUTE against @noisy
	rules := []model.RuleSummary{{
		RuleID:   userRule,
		UserID:   primitive.NewObjectID(),
		Type:     model.RuleTypeActor,
		Action:   model.RuleActionMute,
		Trigger:  "https://example.com/@noisy",
		MatchKey: "ACTOR:https://example.com/@noisy",
	}}

	// Item 1: sender muted by CURRENT rules, clean at receive time -- the "blocked-after-storage"
	// DM case. It also carries a forged reserved key that must be scrubbed.
	later := model.NewInboxActivity()
	later.ActorID = "https://example.com/@noisy"
	later.RawActivity = mapof.Any{"type": "Create", model.PropertyEmissaryLabels: "forged"}

	// Item 2: sender clean now, but stamped BLOCKED at receive time -- the stored-MLS case.
	// The persisted stamp survives even though the rule no longer exists.
	stamped := model.NewInboxActivity()
	stamped.ActorID = "https://example.com/@reformed"
	stamped.RawActivity = mapof.Any{"type": "Create"}
	stamped.Disposition = model.RuleDisposition{Action: model.RuleActionBlock, Tier: model.RuleOriginUser}

	// Item 3: clean both then and now
	clean := model.NewInboxActivity()
	clean.ActorID = "https://example.com/@friend"
	clean.RawActivity = mapof.Any{"type": "Create"}

	result := labeledChunkJSON([]model.InboxActivity{later, stamped, clean}, rules, now)
	require.Len(t, result, 3)

	// Item 1: hidden by the CURRENT mute, forged key gone
	labels1, ok := result[0][model.PropertyEmissaryLabels].([]mapof.Any)
	require.True(t, ok)
	require.Equal(t, "Muted by your rules", labels1[0]["value"])
	require.Equal(t, true, labels1[0]["isHidden"])

	// Item 2: hidden by the PERSISTED stamp
	labels2, ok := result[1][model.PropertyEmissaryLabels].([]mapof.Any)
	require.True(t, ok)
	require.Equal(t, "Blocked by your rules", labels2[0]["value"])

	// Item 3: no labels at all
	require.NotContains(t, result[2], model.PropertyEmissaryLabels)
}

// An expired current rule contributes nothing, but the persisted stamp still serves.
func TestInbox_LabeledChunkJSON_ExpiredRule(t *testing.T) {

	now := int64(1_700_000_000)

	rules := []model.RuleSummary{{
		RuleID:     primitive.NewObjectID(),
		UserID:     primitive.NewObjectID(),
		Type:       model.RuleTypeActor,
		Action:     model.RuleActionMute,
		MatchKey:   "ACTOR:https://example.com/@sender",
		ExpireDate: now - 1000,
	}}

	item := model.NewInboxActivity()
	item.ActorID = "https://example.com/@sender"
	item.RawActivity = mapof.Any{"type": "Create"}

	result := labeledChunkJSON([]model.InboxActivity{item}, rules, now)
	require.NotContains(t, result[0], model.PropertyEmissaryLabels)
}
