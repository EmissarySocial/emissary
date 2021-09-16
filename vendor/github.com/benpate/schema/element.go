package schema

import (
	"encoding/json"
	"strings"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Element interface wraps all of the methods required for schema elements.
type Element interface {

	// Type returns the Type of this particular schema element
	Type() Type

	// Validate checks an arbitrary data structure against the rules in the schema
	Validate(interface{}) error

	// Path traverses this schema to find child element that matches the provided path
	Path(path.Path) (Element, error)

	// MarshalMap populates the object data into a map[string]interface{}
	MarshalMap() map[string]interface{}

	// MarshalJavascript generates Javascript validation code for this element type.
	MarshalJavascript(*strings.Builder)
}

// WritableElement represents an Element (usually a pointer to a concrete type) whose value can be changed.
type WritableElement interface {

	// UnmarshalMap tries to populate this object using data from a map[string]interface{}
	UnmarshalMap(map[string]interface{}) error

	Element
}

// UnmarshalJSON tries to parse a []byte into a schema.Element
func UnmarshalJSON(data []byte) (Element, error) {

	var result map[string]interface{}

	if err := json.Unmarshal(data, &result); err != nil {
		derp.Report(err)
		return nil, derp.Wrap(err, "schema.UnmarshalJSON", "Error unmarshalling JSON", string(data))
	}

	element, err := UnmarshalMap(result)

	if err != nil {
		return nil, derp.Wrap(err, "schema.UnmarshalJSON", "Error unmarshalling map", string(data))
	}

	return element, nil
}

// UnmarshalMap tries to parse a map[string]interface{} into a schema.Element
func UnmarshalMap(data interface{}) (Element, error) {

	var result WritableElement

	if data == nil {
		return nil, derp.New(500, "schema.UnmarshalElement", "Element is nil")
	}

	dataMap, ok := data.(map[string]interface{})

	if !ok {
		return nil, derp.New(500, "schema.UnmarshalElement", "data is not map[string]interface{}", data)
	}

	switch Type(convert.String(dataMap["type"])) {

	case TypeAny:
		result = &Any{}

	case TypeArray:
		result = &Array{}

	case TypeBoolean:
		result = &Boolean{}

	case TypeInteger:
		result = &Integer{}

	case TypeNumber:
		result = &Number{}

	case TypeObject:
		result = &Object{}

	case TypeString:
		result = &String{}

	default:
		return nil, derp.New(500, "schema.UnmarshalElement", "Unrecognized data type", data)

	}

	err := result.UnmarshalMap(dataMap)
	return result, err

}
