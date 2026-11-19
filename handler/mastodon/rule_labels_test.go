package mastodon

import (
	"encoding/json"
	"testing"

	"github.com/benpate/hannibal/metadata"
	"github.com/stretchr/testify/require"
)

// TestLabelFilterResults confirms annotating labels become "warn" filter results and hiding ones are left out
func TestLabelFilterResults(t *testing.T) {

	require.Empty(t, labelFilterResults(nil))

	labels := metadata.LabelSet{
		{Value: "Politics", Href: "https://my.example/@me/rules/1"},
		{Value: "Muted by your rules", IsHidden: true},
		{Value: "Spoilers"},
		{Value: ""},
	}

	results := labelFilterResults(labels)
	require.Len(t, results, 2, "the hiding label and the empty one are left out")

	require.Equal(t, "Politics", results[0].Filter.Title)
	require.Equal(t, "https://my.example/@me/rules/1", results[0].Filter.ID)
	require.Equal(t, "warn", results[0].Filter.FilterAction)
	require.Contains(t, results[0].Filter.Context, "home")

	require.Equal(t, "label:Spoilers", results[1].Filter.ID, "a label with no link still gets a stable ID")
}

// TestLabelFilterResults_JSON confirms the encoded result decodes in the official app, which reads
// keywords as a list and expires_at as an optional date
func TestLabelFilterResults_JSON(t *testing.T) {

	encoded, err := json.Marshal(labelFilterResults(metadata.LabelSet{{Value: "Politics"}})[0])
	require.NoError(t, err)

	var decoded map[string]map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	filter := decoded["filter"]
	require.Equal(t, []any{}, filter["keywords"])
	require.Nil(t, filter["expires_at"])
	require.Equal(t, "warn", filter["filter_action"])
}
