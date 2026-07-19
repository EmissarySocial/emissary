package model

import (
	"encoding/json"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestInboxActivity_LabeledJSON pins the SSE payload contract: labels applied from the stored
// Disposition, forged keys scrubbed, and the stored RawActivity never modified.
func TestInboxActivity_LabeledJSON(t *testing.T) {

	activity := NewInboxActivity()
	activity.RawActivity = mapof.Any{
		"type":                 "Create",
		PropertyEmissaryLabels: "forged",
	}
	activity.Disposition = RuleDisposition{Action: RuleActionBlock, Tier: RuleOriginAdmin}

	payload := mapof.Any{}
	require.NoError(t, json.Unmarshal([]byte(activity.LabeledJSON()), &payload))

	// The payload carries the server-generated labels, not the forged value
	labels, ok := payload[PropertyEmissaryLabels].([]any)
	require.True(t, ok)
	require.Len(t, labels, 1)
	require.Equal(t, "Blocked by server policy", labels[0].(map[string]any)["value"])
	require.Equal(t, true, labels[0].(map[string]any)["isHidden"])

	// The stored RawActivity was not touched
	require.Equal(t, "forged", activity.RawActivity[PropertyEmissaryLabels])
}

// A clean disposition scrubs the reserved key and adds nothing.
func TestInboxActivity_LabeledJSON_Clean(t *testing.T) {

	activity := NewInboxActivity()
	activity.RawActivity = mapof.Any{
		"type":                 "Create",
		PropertyEmissaryLabels: "forged",
	}

	payload := mapof.Any{}
	require.NoError(t, json.Unmarshal([]byte(activity.LabeledJSON()), &payload))

	require.NotContains(t, payload, PropertyEmissaryLabels)
	require.Equal(t, "Create", payload["type"])
}

// A nil RawActivity never panics, even when a disposition wants to write labels.
func TestInboxActivity_LabeledJSON_NilRawActivity(t *testing.T) {

	activity := NewInboxActivity()
	activity.Disposition = RuleDisposition{Action: RuleActionMute, Tier: RuleOriginUser}

	payload := mapof.Any{}
	require.NoError(t, json.Unmarshal([]byte(activity.LabeledJSON()), &payload))
	require.Contains(t, payload, PropertyEmissaryLabels)
}
