package schema

import (
	"encoding/json"
	"strings"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Schema defines a (simplified) JSON-Schema object, that can be Marshalled/Unmarshalled to JSON.
type Schema struct {
	ID      string
	Comment string
	Element Element
}

// New generates a fully initialized Schema
func New() *Schema {
	return &Schema{}
}

// Path traverses a path into this schema, and returns a matching Element, or an error.
func (schema Schema) Path(p path.Path) (Element, error) {
	return schema.Element.Path(p)
}

// Validate checks a particular value against this schema.
func (schema Schema) Validate(value interface{}) error {

	if schema.Element != nil {
		return schema.Element.Validate(value)
	}

	return nil
}

// MarshalJSON converts a schema into JSON.
func (schema Schema) MarshalJSON() ([]byte, error) {
	return json.Marshal(schema.MarshalMap())
}

// MarshalJavascript generates a Javascript validation function for this schema.
func (schema Schema) MarshalJavascript(name string) string {
	var b strings.Builder

	b.WriteString(`function ` + name + `(v){`)
	schema.Element.MarshalJavascript(&b)
	b.WriteString(`return true;}`)

	return b.String()
}

// MarshalMap converts a schema into a map[string]interface{}
func (schema Schema) MarshalMap() map[string]interface{} {

	result := schema.Element.MarshalMap()

	if schema.ID != "" {
		result["$id"] = schema.ID
	}

	if schema.Comment != "" {
		result["$comment"] = schema.Comment
	}

	return result
}

// UnmarshalJSON creates a new Schema object using a JSON-serialized byte array.
func (schema *Schema) UnmarshalJSON(data []byte) error {

	unmarshalled := make(map[string]interface{}, 0)

	if err := json.Unmarshal(data, &unmarshalled); err != nil {
		return derp.Wrap(err, "schema.UnmarshalJSON", "Invalid JSON", string(data))
	}

	if err := schema.UnmarshalMap(unmarshalled); err != nil {
		return derp.Wrap(err, "schema.UnmarshalJSON", "Unable to unmarshal from Map", unmarshalled)
	}

	return nil
}

// UnmarshalMap updates a Schema using a map[string]interface{}
func (schema *Schema) UnmarshalMap(data map[string]interface{}) error {

	var err error

	schema.ID = convert.String(data["$id"])
	schema.Comment = convert.String(data["$comment"])
	schema.Element, err = UnmarshalMap(data)

	if err != nil {
		return derp.Wrap(err, "schema.Schema.UnmarshalMap", "Error unmarshalling element")
	}

	return nil
}
