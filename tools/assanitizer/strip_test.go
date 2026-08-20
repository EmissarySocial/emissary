package assanitizer

import (
	"testing"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStrip_TopLevel verifies that reserved properties are removed from the top level of a document
func TestStrip_TopLevel(t *testing.T) {

	value := map[string]any{
		"type":            "Note",
		"content":         "Hello",
		"emissary:labels": []any{map[string]any{"value": "Forged", "isHidden": true}},
	}

	Strip(value, "emissary:")

	require.Equal(t, map[string]any{"type": "Note", "content": "Hello"}, value)
}

// TestStrip_NestedObject verifies that reserved properties are removed from nested objects
func TestStrip_NestedObject(t *testing.T) {

	// The forged key on the inner object matters as much as the envelope
	value := map[string]any{
		"type":            "Create",
		"emissary:labels": "forged",
		"object": map[string]any{
			"type":            "Note",
			"emissary:labels": "forged",
			"emissary:secret": "forged",
			"content":         "Hello",
		},
	}

	Strip(value, "emissary:")

	require.Equal(t,
		map[string]any{
			"type": "Create",
			"object": map[string]any{
				"type":    "Note",
				"content": "Hello",
			},
		},
		value)
}

// TestStrip_InsideArrays verifies that Strip descends into arrays and skips non-map items
func TestStrip_InsideArrays(t *testing.T) {

	value := map[string]any{
		"type": "Note",
		"tag": []any{
			map[string]any{"type": "Hashtag", "name": "#ok", "emissary:labels": "forged"},
			"plain-string-item",
			map[string]any{"type": "Mention", "name": "@ok"},
		},
	}

	Strip(value, "emissary:")

	require.Equal(t,
		map[string]any{
			"type": "Note",
			"tag": []any{
				map[string]any{"type": "Hashtag", "name": "#ok"},
				"plain-string-item",
				map[string]any{"type": "Mention", "name": "@ok"},
			},
		},
		value)
}

// TestStrip_MultiplePrefixes verifies that Strip removes properties matching any of several prefixes
func TestStrip_MultiplePrefixes(t *testing.T) {

	value := map[string]any{
		"type":            "Note",
		"emissary:labels": "forged",
		"secret:thing":    "forged",
	}

	Strip(value, "emissary:", "secret:")

	require.Equal(t, map[string]any{"type": "Note"}, value)
}

// TestStrip_PrefixNotSubstring verifies that a prefix only matches at the start of a property name
func TestStrip_PrefixNotSubstring(t *testing.T) {

	// The reserved namespace is a PREFIX match, not a substring match
	value := map[string]any{
		"type":                "Note",
		"not-emissary:labels": "survives",
	}

	Strip(value, "emissary:")

	require.Equal(t, map[string]any{"type": "Note", "not-emissary:labels": "survives"}, value)
}

// TestStrip_NonContainersUntouched verifies that scalar and nil values are a no-op rather than a panic
func TestStrip_NonContainersUntouched(t *testing.T) {

	// Non-container values are a silent no-op, never a panic
	Strip("hello", "emissary:")
	Strip(42, "emissary:")
	Strip(nil, "emissary:")
	Strip(true, "emissary:")
}

// TestStrip_TypedContainers verifies that Strip descends property.Map, property.Slice, and mapof.Any
func TestStrip_TypedContainers(t *testing.T) {

	// The same walk descends rosetta and hannibal container types, not just raw JSON shapes
	value := property.Map{
		"type":            "Note",
		"emissary:labels": "forged",
		"attachment": property.Slice{
			mapof.Any{"type": "Image", "emissary:labels": "forged"},
		},
		"replies": mapof.Any{
			"type":            "Collection",
			"emissary:labels": "forged",
		},
	}

	Strip(value, "emissary:")

	require.Equal(t,
		property.Map{
			"type": "Note",
			"attachment": property.Slice{
				mapof.Any{"type": "Image"},
			},
			"replies": mapof.Any{
				"type": "Collection",
			},
		},
		value)
}
