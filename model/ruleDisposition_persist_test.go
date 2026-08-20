package model

import (
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// dispositionCarrier mirrors how InboxActivity embeds a disposition, so the tests pin the
// omitempty + Zeroer mechanics without dragging the whole InboxActivity along.
type dispositionCarrier struct {
	Disposition RuleDisposition `bson:"disposition,omitempty"`
}

// TestRuleDisposition_IsZero verifies which disposition values count as "clean"
func TestRuleDisposition_IsZero(t *testing.T) {

	require.True(t, RuleDisposition{}.IsZero())
	require.False(t, RuleDisposition{Action: RuleActionMute}.IsZero())
	require.False(t, RuleDisposition{Tier: RuleOriginUser}.IsZero())
	require.False(t, RuleDisposition{RuleID: primitive.NewObjectID()}.IsZero())
	require.False(t, RuleDisposition{Labels: []RuleLabelMatch{{Label: "Politics"}}}.IsZero())
}

// A clean disposition writes NO field at all -- which is also why pre-4C rows need no migration:
// a missing field unmarshals to the zero disposition.
func TestRuleDisposition_BSONOmitsClean(t *testing.T) {

	data, err := bson.Marshal(dispositionCarrier{})
	require.NoError(t, err)

	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(data, &raw))
	require.NotContains(t, raw, "disposition")
}

// A dirty disposition round-trips losslessly through bson.
func TestRuleDisposition_BSONRoundTrip(t *testing.T) {

	original := dispositionCarrier{
		Disposition: RuleDisposition{
			Action: RuleActionBlock,
			Tier:   RuleOriginAdmin,
			RuleID: primitive.NewObjectID(),
			Labels: []RuleLabelMatch{
				{RuleID: primitive.NewObjectID(), Source: "@blocklist@example.com", Label: "Spam"},
			},
		},
	}

	data, err := bson.Marshal(original)
	require.NoError(t, err)

	// The field is present when dirty...
	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(data, &raw))
	require.Contains(t, raw, "disposition")

	// ...and unmarshals back without loss
	restored := dispositionCarrier{}
	require.NoError(t, bson.Unmarshal(data, &restored))
	require.Equal(t, original, restored)
}

// TestRuleDisposition_Merge verifies that merging two dispositions keeps the higher severity
func TestRuleDisposition_Merge(t *testing.T) {

	blockedAdmin := RuleDisposition{Action: RuleActionBlock, Tier: RuleOriginAdmin, RuleID: primitive.NewObjectID()}
	mutedUser := RuleDisposition{Action: RuleActionMute, Tier: RuleOriginUser, RuleID: primitive.NewObjectID()}
	mutedAdmin := RuleDisposition{Action: RuleActionMute, Tier: RuleOriginAdmin, RuleID: primitive.NewObjectID()}
	clean := RuleDisposition{}

	// Clean + clean = clean
	require.True(t, clean.Merge(clean).IsZero())

	// The other side's severity wins when greater
	require.Equal(t, RuleActionMute, clean.Merge(mutedUser).Action)
	require.Equal(t, RuleActionBlock, mutedUser.Merge(blockedAdmin).Action)
	require.Equal(t, blockedAdmin.RuleID, mutedUser.Merge(blockedAdmin).RuleID)

	// The receiver's severity stands when greater
	require.Equal(t, RuleActionBlock, blockedAdmin.Merge(mutedUser).Action)
	require.Equal(t, blockedAdmin.RuleID, blockedAdmin.Merge(mutedUser).RuleID)

	// Ties keep the RECEIVER's attribution (current.Merge(persisted): the live rule wins)
	tied := mutedUser.Merge(mutedAdmin)
	require.Equal(t, RuleOriginUser, tied.Tier)
	require.Equal(t, mutedUser.RuleID, tied.RuleID)
}

// TestRuleDisposition_MergeLabels verifies that labels from both dispositions are combined
func TestRuleDisposition_MergeLabels(t *testing.T) {

	shared := primitive.NewObjectID()
	onlyMine := RuleLabelMatch{RuleID: primitive.NewObjectID(), Label: "Politics"}
	onlyTheirs := RuleLabelMatch{RuleID: primitive.NewObjectID(), Label: "Sports"}

	mine := RuleDisposition{Labels: []RuleLabelMatch{onlyMine, {RuleID: shared, Label: "News"}}}
	theirs := RuleDisposition{Labels: []RuleLabelMatch{onlyTheirs, {RuleID: shared, Label: "News"}}}

	merged := mine.Merge(theirs)

	// Union, deduplicated by RuleID: the shared rule appears once
	require.Len(t, merged.Labels, 3)
	require.Contains(t, merged.Labels, onlyMine)
	require.Contains(t, merged.Labels, onlyTheirs)

	// Neither input was modified
	require.Len(t, mine.Labels, 2)
	require.Len(t, theirs.Labels, 2)
}

// TestRuleDisposition_ApplyLabels verifies that a disposition writes its labels into a document, scrubbing any forged ones
func TestRuleDisposition_ApplyLabels(t *testing.T) {

	// RULE: a forged value under the reserved key is always scrubbed, even by a clean disposition
	target := mapof.Any{
		"type":                 "Note",
		PropertyEmissaryLabels: "forged",
	}

	RuleDisposition{}.ApplyLabels(target)
	require.NotContains(t, target, PropertyEmissaryLabels)
	require.Equal(t, "Note", target["type"])

	// A filtering disposition writes the hidden label first, then annotations
	disposition := RuleDisposition{
		Action: RuleActionMute,
		Tier:   RuleOriginUser,
		Labels: []RuleLabelMatch{{RuleID: primitive.NewObjectID(), Label: "Politics"}},
	}

	disposition.ApplyLabels(target)

	labels, ok := target[PropertyEmissaryLabels].([]mapof.Any)
	require.True(t, ok)
	require.Len(t, labels, 2)
	require.Equal(t, "Muted by your rules", labels[0]["value"])
	require.Equal(t, true, labels[0]["isHidden"])
	require.Equal(t, "Politics", labels[1]["value"])
	require.Equal(t, false, labels[1]["isHidden"])

	// Href is omitted while empty (nothing populates it yet)
	require.NotContains(t, labels[0], "href")
}
