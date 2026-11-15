package assanitizer

import (
	"encoding/json"
	"testing"

	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TestWithoutBlindRecipients verifies that `bto` and `bcc` are removed at every level, that
// the copy keeps its types, and that the original is unchanged
func TestWithoutBlindRecipients(t *testing.T) {

	build := func() mapof.Any {
		return mapof.Any{
			"type": "Create",
			"to":   []any{"https://example.com/@alice"},
			"bto":  []any{"https://example.com/@bob"},
			"bcc":  "https://example.com/@carol",
			"object": map[string]any{
				"type": "Note",
				"bcc":  []any{"https://example.com/@dave"},
				"tag":  []any{mapof.Any{"name": "go", "bto": "https://example.com/@erin"}},
			},
		}
	}

	original := build()

	result, err := WithoutBlindRecipients(original)
	require.NoError(t, err)

	require.Equal(t, mapof.Any{
		"type": "Create",
		"to":   []any{"https://example.com/@alice"},
		"object": map[string]any{
			"type": "Note",
			"tag":  []any{mapof.Any{"name": "go"}},
		},
	}, result)

	// Changing the copy leaves the original alone, and so did the strip
	result["object"].(map[string]any)["type"] = "changed"
	require.Equal(t, build(), original)
}

// TestWithoutBlindRecipients_DatabaseTypes verifies that an activity read from MongoDB is
// stripped inside the driver's own document and array types, which keep their types
func TestWithoutBlindRecipients_DatabaseTypes(t *testing.T) {

	objectID := primitive.NewObjectID()

	original := mapof.Any{
		"_id":       objectID,
		"published": primitive.DateTime(1700000000000),
		"bcc":       primitive.A{"https://example.com/@carol"},
		"object": primitive.M{
			"bto":        primitive.A{"https://example.com/@bob"},
			"attachment": primitive.A{primitive.M{"bcc": "https://example.com/@dave", "name": "a"}},
		},
	}

	result, err := WithoutBlindRecipients(original)
	require.NoError(t, err)

	require.Equal(t, mapof.Any{
		"_id":       objectID,
		"published": primitive.DateTime(1700000000000),
		"object": primitive.M{
			"attachment": primitive.A{primitive.M{"name": "a"}},
		},
	}, result)

	require.Contains(t, original, "bcc")
	require.Contains(t, original["object"], "bto")
}

// TestWithoutBlindRecipients_Nil verifies that a nil activity stays nil
func TestWithoutBlindRecipients_Nil(t *testing.T) {

	result, err := WithoutBlindRecipients(mapof.Any(nil))

	require.NoError(t, err)
	require.Nil(t, result)
}

// TestWithoutBlindRecipients_Refuses verifies that a value which could hide a blind recipient
// is an error, that nothing is returned, and that the error does not repeat the value
func TestWithoutBlindRecipients_Refuses(t *testing.T) {

	type hidden struct {
		BCC []string `json:"bcc"`
	}

	pointed := map[string]any{"bcc": "https://example.com/@blind"}

	selfReference := mapof.Any{}
	selfReference["object"] = selfReference

	values := map[string]mapof.Any{
		"struct":          {"object": hidden{BCC: []string{"https://example.com/@blind"}}},
		"pointer":         {"object": &pointed},
		"raw JSON":        {"object": json.RawMessage(`{"bcc":"https://example.com/@blind"}`)},
		"primitive.D":     {"object": primitive.D{{Key: "bcc", Value: "https://example.com/@blind"}}},
		"contains itself": selfReference,
		"function":        {"object": func() {}},
	}

	for name, value := range values {
		t.Run(name, func(t *testing.T) {

			result, err := WithoutBlindRecipients(value)

			require.Error(t, err)
			require.Nil(t, result)
			require.NotContains(t, err.Error(), "@blind")
		})
	}
}
