package convert

type Maplike interface {
	AsMapOfInterface() map[string]interface{}
}

// MapOfInterface attempts to convert the generic value into a map[string]interface{}
// The boolean result value returns TRUE if successful.  FALSE otherwise
func MapOfInterface(value interface{}) (map[string]interface{}, bool) {

	switch v := value.(type) {

	case map[string]interface{}:
		return v, true

	case Maplike:
		return v.AsMapOfInterface(), true

	}

	// Fall through means conversion failed
	return make(map[string]interface{}), false
}
