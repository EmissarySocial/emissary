package assanitizer

import (
	"testing"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

func TestStrip_TopLevel(t *testing.T) {

	value := map[string]any{
		"type":            "Note",
		"content":         "Hello",
		"emissary:labels": []any{map[string]any{"value": "Forged", "isHidden": true}},
	}

	Strip(value, "emissary:")

	require.Equal(t, map[string]any{"type": "Note", "content": "Hello"}, value)
}

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

func TestStrip_MultiplePrefixes(t *testing.T) {

	value := map[string]any{
		"type":            "Note",
		"emissary:labels": "forged",
		"secret:thing":    "forged",
	}

	Strip(value, "emissary:", "secret:")

	require.Equal(t, map[string]any{"type": "Note"}, value)
}

func TestStrip_PrefixNotSubstring(t *testing.T) {

	// The reserved namespace is a PREFIX match, not a substring match
	value := map[string]any{
		"type":                "Note",
		"not-emissary:labels": "survives",
	}

	Strip(value, "emissary:")

	require.Equal(t, map[string]any{"type": "Note", "not-emissary:labels": "survives"}, value)
}

func TestStrip_NonContainersUntouched(t *testing.T) {

	// Non-container values are a silent no-op, never a panic
	Strip("hello", "emissary:")
	Strip(42, "emissary:")
	Strip(nil, "emissary:")
	Strip(true, "emissary:")
}

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
