package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRuleDisposition_LabelSet_Block confirms a block yields a leading hidden Label, attributed to
// the server for an admin-tier rule.
func TestRuleDisposition_LabelSet_Block(t *testing.T) {

	labels := RuleDisposition{Action: RuleActionBlock, Tier: RuleOriginAdmin}.LabelSet()

	require.Len(t, labels, 1)
	require.True(t, labels[0].IsHidden)
	require.Equal(t, "Blocked by server policy", labels[0].Value)
	require.True(t, labels.IsHidden())
}

// TestRuleDisposition_LabelSet_Mute confirms a mute also hides, attributed to the viewer.
func TestRuleDisposition_LabelSet_Mute(t *testing.T) {

	labels := RuleDisposition{Action: RuleActionMute, Tier: RuleOriginUser}.LabelSet()

	require.Len(t, labels, 1)
	require.True(t, labels[0].IsHidden)
	require.Equal(t, "Muted by your rules", labels[0].Value)
}

// TestRuleDisposition_LabelSet_Clean confirms a disposition with no matches yields an empty set.
func TestRuleDisposition_LabelSet_Clean(t *testing.T) {
	require.Empty(t, RuleDisposition{}.LabelSet())
}

// TestRuleDisposition_LabelSet_Labels confirms LABEL matches ride as annotations after any hidden
// Label (hidden-first), that empty label text is skipped, and that a labels-only disposition hides
// nothing.
func TestRuleDisposition_LabelSet_Labels(t *testing.T) {

	labels := RuleDisposition{
		Action: RuleActionBlock,
		Tier:   RuleOriginUser,
		Labels: []RuleLabelMatch{
			{Label: "Spam"},
			{Label: ""}, // skipped
			{Label: "Politics"},
		},
	}.LabelSet()

	require.Len(t, labels, 3)
	require.True(t, labels[0].IsHidden) // hidden first
	require.Equal(t, "Blocked by your rules", labels[0].Value)
	require.False(t, labels[1].IsHidden)
	require.Equal(t, "Spam", labels[1].Value)
	require.Equal(t, "Politics", labels[2].Value)

	// A labels-only disposition (no filtering action) hides nothing.
	labelsOnly := RuleDisposition{Labels: []RuleLabelMatch{{Label: "Politics"}}}.LabelSet()
	require.False(t, labelsOnly.IsHidden())
	require.True(t, labelsOnly.HasAnnotations())
}
