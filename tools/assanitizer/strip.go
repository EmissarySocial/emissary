package assanitizer

import (
	"strings"

	"github.com/benpate/hannibal/property"
	"github.com/benpate/rosetta/mapof"
)

// Strip removes every property whose name begins with one of the reserved prefixes from a JSON
// value, recursing into nested objects and arrays. Containers are modified IN PLACE; values that
// are not JSON containers are left untouched.
func Strip(value any, prefixes ...string) {

	switch typed := value.(type) {

	case map[string]any:
		stripMap(typed, prefixes)

	case mapof.Any:
		stripMap(typed, prefixes)

	case property.Map:
		stripMap(typed, prefixes)

	case []any:
		for _, item := range typed {
			Strip(item, prefixes...)
		}

	case property.Slice:
		for _, item := range typed {
			Strip(item, prefixes...)
		}

	case []map[string]any:
		for _, item := range typed {
			stripMap(item, prefixes)
		}

	case []mapof.Any:
		for _, item := range typed {
			stripMap(item, prefixes)
		}
	}
}

// stripMap removes reserved keys from a single map, then recurses into the surviving values.
func stripMap(value map[string]any, prefixes []string) {

	for key, child := range value {

		// Reserved keys are deleted outright. (Deleting during range is safe in Go.)
		if isReserved(key, prefixes) {
			delete(value, key)
			continue
		}

		// Surviving children may hide reserved keys deeper down.
		Strip(child, prefixes...)
	}
}

// isReserved returns TRUE if the key begins with any of the reserved prefixes.
func isReserved(key string, prefixes []string) bool {

	for _, prefix := range prefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}
