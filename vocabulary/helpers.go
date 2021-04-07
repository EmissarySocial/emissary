package vocabulary

import (
	"github.com/benpate/path"
	"github.com/benpate/schema"
)

// locateSchema looks up schema and values using a variable path.
func locateSchema(pathString string, original *schema.Schema, value interface{}) (schema.Element, interface{}) {

	var resultSchema schema.Element
	var resultValue interface{}

	resultSchema = schema.Any{}

	// If the path is empty, then return empty values
	if pathString != "" {

		// Parse the path to the field value.
		pathObject := path.New(pathString)

		if s, err := original.Path(pathObject); err == nil {
			resultSchema = s
		}

		if value, err := pathObject.Get(value); err == nil {
			resultValue = value
		}
	}

	return resultSchema, resultValue
}
