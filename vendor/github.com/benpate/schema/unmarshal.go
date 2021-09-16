package schema

import "encoding/json"

// Unmarshal converts a JSON string into a schema.  If the
// string cannot be converted, then an empty schema is returned.
func Unmarshal(original string) Schema {

	var result Schema

	json.Unmarshal([]byte(original), &result)

	return result
}
