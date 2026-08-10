package assanitizer

import (
	"slices"
	"strings"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/rosetta/mapof"
)

// Strip removes every property whose name begins with one of the reserved prefixes from a JSON
// value, recursing into nested objects and arrays. Containers are modified IN PLACE; values that
// are not JSON containers are left untouched.
func Strip(value any, prefixes ...string) {
	stripValue(value, func(key string) bool {
		return slices.ContainsFunc(prefixes, func(prefix string) bool {
			return strings.HasPrefix(key, prefix)
		})
	})
}

// StripKeys removes every property whose name EXACTLY matches one of the given keys, recursing
// into nested objects and arrays. Containers are modified IN PLACE, so DeepCopy first if anything
// upstream still reads the original.
//
// Use this for well-known AS2 property names (bto, bcc); use Strip for reserved namespace
// prefixes. The distinction matters: prefix matching on a bare name like "bto" would also claim
// any future property that merely starts with those letters.
func StripKeys(value any, keys ...string) {
	stripValue(value, func(key string) bool {
		return slices.Contains(keys, key)
	})
}

// stripValue walks a JSON value, deleting every key that `match` selects. Containers are modified
// IN PLACE; values that are not JSON containers are left untouched.
func stripValue(value any, match func(string) bool) {

	switch typed := value.(type) {

	case map[string]any:
		stripMap(typed, match)

	case mapof.Any:
		stripMap(typed, match)

	case property.Map:
		stripMap(typed, match)

	case []any:
		for _, item := range typed {
			stripValue(item, match)
		}

	case property.Slice:
		for _, item := range typed {
			stripValue(item, match)
		}

	case []map[string]any:
		for _, item := range typed {
			stripMap(item, match)
		}

	case []mapof.Any:
		for _, item := range typed {
			stripMap(item, match)
		}
	}
}

// stripMap removes matching keys from a single map, then recurses into the surviving values.
func stripMap(value map[string]any, match func(string) bool) {

	for key, child := range value {

		// Matching keys are deleted outright. (Deleting during range is safe in Go.)
		if match(key) {
			delete(value, key)
			continue
		}

		// Surviving children may hide matching keys deeper down.
		stripValue(child, match)
	}
}
