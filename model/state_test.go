package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
)

// TestState_IsOutbox_JSON confirms that a Template's "outbox" value reaches the State, and is FALSE when absent
func TestState_IsOutbox_JSON(t *testing.T) {

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "declared true", input: `{"label":"Published", "outbox":true}`, expected: true},
		{name: "declared false", input: `{"label":"Draft", "outbox":false}`, expected: false},
		{name: "not declared", input: `{"label":"Draft"}`, expected: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := NewState()
			require.NoError(t, json.Unmarshal([]byte(test.input), &state))
			require.Equal(t, test.expected, state.IsOutbox)
		})
	}
}

// TestState_IsOutbox_BSON confirms that the flag survives a round trip through BSON
func TestState_IsOutbox_BSON(t *testing.T) {

	original := State{StateID: "published", Label: "Published", IsOutbox: true}

	encoded, err := bson.Marshal(original)
	require.NoError(t, err)

	decoded := NewState()
	require.NoError(t, bson.Unmarshal(encoded, &decoded))
	require.Equal(t, original, decoded)

	// The stored key is "outbox"
	raw := bson.M{}
	require.NoError(t, bson.Unmarshal(encoded, &raw))
	require.Equal(t, true, raw["outbox"])
}

// TestState_IsOutbox_JSONKey confirms that the flag is written as "outbox"
func TestState_IsOutbox_JSONKey(t *testing.T) {

	encoded, err := json.Marshal(State{StateID: "published", IsOutbox: true})
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"outbox":true`)
	require.NotContains(t, string(encoded), "isOutbox")
}

// TestTemplate_States_IsOutbox confirms that each State in a Template's "states" block sets the flag on its own
func TestTemplate_States_IsOutbox(t *testing.T) {

	template := NewTemplate("test", nil)

	input := `{"states":{"default":{"label":"Draft"}, "published":{"label":"Published", "outbox":true}, "archived":{"label":"Archived"}}}`
	require.NoError(t, json.Unmarshal([]byte(input), &template))

	require.False(t, template.States["default"].IsOutbox)
	require.True(t, template.States["published"].IsOutbox)
	require.False(t, template.States["archived"].IsOutbox)
}
