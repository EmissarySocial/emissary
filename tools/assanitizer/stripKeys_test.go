package assanitizer

import (
	"testing"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/hannibal/vocab"
	"github.com/benpate/rosetta/mapof"
	"github.com/stretchr/testify/require"
)

// TestStripKeys_ExactMatch verifies that StripKeys matches whole property names, and does NOT
// behave like Strip's prefix matcher -- the specific reason it exists as a separate function.
func TestStripKeys_ExactMatch(t *testing.T) {

	value := mapof.Any{
		vocab.PropertyTo:  "https://example.com/users/visible",
		vocab.PropertyBTo: "https://example.com/users/secret",
		vocab.PropertyBCC: "https://example.com/users/alsoSecret",
		"bccPolicy":       "keep-me",
		"btoken":          "keep-me-too",
	}

	StripKeys(value, vocab.PropertyBTo, vocab.PropertyBCC)

	require.NotContains(t, value, vocab.PropertyBTo)
	require.NotContains(t, value, vocab.PropertyBCC)

	// A prefix matcher would have eaten these.
	require.Equal(t, "keep-me", value["bccPolicy"])
	require.Equal(t, "keep-me-too", value["btoken"])
	require.Equal(t, "https://example.com/users/visible", value[vocab.PropertyTo])
}

// TestStripKeys_Nested verifies the walk reaches blind addressing on a wrapped object -- a Create
// carrying a Note that carries its own bcc.
func TestStripKeys_Nested(t *testing.T) {

	value := mapof.Any{
		vocab.PropertyType: vocab.ActivityTypeCreate,
		vocab.PropertyBCC:  []any{"https://example.com/users/secret"},
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType: vocab.ObjectTypeNote,
			vocab.PropertyBTo:  "https://example.com/users/deeperSecret",
			vocab.PropertyTag: []any{
				map[string]any{
					vocab.PropertyType: vocab.LinkTypeMention,
					vocab.PropertyBCC:  "https://example.com/users/deepestSecret",
				},
			},
		},
	}

	StripKeys(value, vocab.PropertyBTo, vocab.PropertyBCC)

	require.NotContains(t, value, vocab.PropertyBCC)

	object := value[vocab.PropertyObject].(map[string]any)
	require.NotContains(t, object, vocab.PropertyBTo)

	tag := object[vocab.PropertyTag].([]any)[0].(map[string]any)
	require.NotContains(t, tag, vocab.PropertyBCC)
}

// TestCloneThenStripKeys_LeavesOriginalIntact is the regression guard for the delivery hazard: the
// source map must still carry full addressing after the copy is stripped, because delivery
// enumeration (CalcRecipients, RangeAddressees) reads it afterwards. This is the composition that
// service.Inbox.Save performs; Clone's own deep-copy contract is tested in hannibal/property.
func TestCloneThenStripKeys_LeavesOriginalIntact(t *testing.T) {

	original := mapof.Any{
		vocab.PropertyTo:  "https://example.com/users/visible",
		vocab.PropertyBCC: []any{"https://example.com/users/secret"},
		vocab.PropertyObject: map[string]any{
			vocab.PropertyType: vocab.ObjectTypeNote,
			vocab.PropertyBTo:  "https://example.com/users/deeperSecret",
		},
	}

	stripped := property.Map(original).Clone().Map()
	StripKeys(stripped, vocab.PropertyBTo, vocab.PropertyBCC)

	// The copy is clean...
	require.NotContains(t, stripped, vocab.PropertyBCC)
	require.NotContains(t, stripped[vocab.PropertyObject].(map[string]any), vocab.PropertyBTo)

	// ...and the original is untouched at every level. A shallow clone would fail the nested half.
	require.Contains(t, original, vocab.PropertyBCC)
	require.Contains(t, original[vocab.PropertyObject].(map[string]any), vocab.PropertyBTo)
}

// TestStrip_PrefixBehaviorUnchanged locks in the original reserved-namespace behavior across the
// predicate refactor.
func TestStrip_PrefixBehaviorUnchanged(t *testing.T) {

	value := mapof.Any{
		vocab.PropertyID:  "https://example.com/note/1",
		"emissary:labels": "forged",
		"object": map[string]any{
			vocab.PropertyType: vocab.ObjectTypeNote,
			"emissary:trust":   "forged",
		},
	}

	Strip(value, "emissary:")

	require.NotContains(t, value, "emissary:labels")
	require.NotContains(t, value["object"].(map[string]any), "emissary:trust")
	require.Equal(t, "https://example.com/note/1", value[vocab.PropertyID])
}
