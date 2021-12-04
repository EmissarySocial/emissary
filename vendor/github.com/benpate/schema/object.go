package schema

import (
	"strings"

	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/path"
)

// Object represents an object data type within a JSON-Schema.
type Object struct {
	Properties    map[string]Element
	RequiredProps []string
	Required      bool
}

// Type returns the data type of this Element
func (object Object) Type() Type {
	return TypeObject
}

// Path returns sub-schemas
func (object Object) Path(p path.Path) (Element, error) {

	if p.IsEmpty() {
		return object, nil
	}

	key := p.Head()

	if property, ok := object.Properties[key]; ok {
		return property.Path(p.Tail())
	}

	return nil, derp.New(500, "schema.Object.GetPath", "Property not defined", object, p)
}

// Validate compares a generic data value using this Schema
func (object Object) Validate(value interface{}) error {

	mapValue, mapOk := convert.MapOfInterface(value)

	if !mapOk {
		return derp.New(500, "schema.Object.Validate", "value must be a map", value)
	}

	result := derp.NewCollector()

	for key, value := range mapValue {
		if schema, ok := object.Properties[key]; ok {
			if errs := schema.Validate(value); errs != nil {
				result.Add(Rollup(errs, key))
			}
		} else {
			delete(mapValue, key)
		}
	}

	for _, propertyName := range object.RequiredProps {

		if isEmpty(mapValue[propertyName]) {
			result.Add(ValidationError{Path: propertyName, Message: "Value is required"})
		}
	}

	return result.Error()
}

// MarshalMap populates object data into a map[string]interface{}
func (object Object) MarshalMap() map[string]interface{} {

	properties := make(map[string]interface{}, len(object.Properties))

	for key, element := range object.Properties {
		properties[key] = element.MarshalMap()
	}

	return map[string]interface{}{
		"type":       object.Type(),
		"properties": properties,
		"required":   object.RequiredProps,
	}
}

// UnmarshalMap tries to populate this object using data from a map[string]interface{}
func (object *Object) UnmarshalMap(data map[string]interface{}) error {

	var err error

	if convert.String(data["type"]) != "object" {
		return derp.New(500, "schema.Object.UnmarshalMap", "Data is not type 'object'", data)
	}

	// Handle "simple" required as a boolean
	if required, ok := data["required"].(bool); ok {
		object.Required = required
	}

	if properties, ok := data["properties"].(map[string]interface{}); ok {

		object.Properties = make(map[string]Element, len(properties))

		for key, value := range properties {

			if propertyMap, ok := value.(map[string]interface{}); ok {

				if _, ok := propertyMap["required"]; !ok && object.Required {
					propertyMap["required"] = true
				}

				if propertyObject, err := UnmarshalMap(propertyMap); err == nil {

					object.Properties[key] = propertyObject
				}
			}
		}
	}

	// Handle "standards" required as an array of strings.
	if required, ok := data["required"].([]interface{}); ok {

		object.RequiredProps = convert.SliceOfString(required)

		for _, name := range object.RequiredProps {

			if property, ok := object.Properties[name]; ok {

				switch p := property.(type) {
				case *Any:
					p.Required = true
				case *Array:
					p.Required = true
				case *Boolean:
					p.Required = true
				case *Integer:
					p.Required = true
				case *Number:
					p.Required = true
				case *Object:
					p.Required = true
				case *String:
					p.Required = true
				}
			}
		}
	}

	return err
}

func (object Object) MarshalJavascript(b *strings.Builder) {

}
