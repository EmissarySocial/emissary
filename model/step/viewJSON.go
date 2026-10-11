package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/translate"
)

// ViewJSON is a Step that writes a JSON document, either from one template expression (Value)
// or from a rules pipeline (Rules) applied to the builder's object
type ViewJSON struct {
	Value       *template.Template
	Schema      schema.Schema
	Rules       translate.Pipeline
	ContentType string
}

// NewViewJSON generates a fully initialized ViewJSON step.
func NewViewJSON(stepInfo mapof.Any) (ViewJSON, error) {

	const location = "build.NewViewJSON"

	value := stepInfo.GetString("value")
	_, hasRules := stepInfo["rules"]

	// RULE: A step uses exactly one of "value" or "rules"
	if (value == "") == !hasRules {
		return ViewJSON{}, derp.Validation("Step requires either a value or rules, but not both")
	}

	contentType := stepInfo.GetString("content-type")

	if contentType == "" {
		contentType = "application/json"
	}

	// Build a document from the rules
	if hasRules {

		rules, err := translate.NewFromMap(convert.SliceOfMap(stepInfo["rules"])...)

		if err != nil {
			return ViewJSON{}, derp.Wrap(err, location, "Parsing rules")
		}

		target, err := newViewJSONSchema(stepInfo["schema"])

		if err != nil {
			return ViewJSON{}, derp.Wrap(err, location, "Parsing schema")
		}

		return ViewJSON{
			Schema:      target,
			Rules:       rules,
			ContentType: contentType,
		}, nil
	}

	// Otherwise, render the "value" expression
	value = "{{" + value + " | json}}"

	if jsonp := stepInfo.GetString("jsonp"); jsonp != "" {
		value = jsonp + "(" + value + ");"
	}

	valueTemplate, err := template.New("").Funcs(FuncMap()).Parse(value)

	if err != nil {
		return ViewJSON{}, derp.Wrap(err, location, "Parsing JSON query template")
	}

	return ViewJSON{
		Value:       valueTemplate,
		ContentType: contentType,
	}, nil
}

// newViewJSONSchema returns the schema that the rules write through: an object whose
// undeclared properties are Any, extended by the schema the Template provides (if any)
func newViewJSONSchema(data any) (schema.Schema, error) {

	const location = "step.newViewJSONSchema"

	result := schema.Object{
		Properties: schema.ElementMap{},
		Wildcard:   schema.Any{},
	}

	// With no schema, every property is allowed
	if data == nil {
		return schema.New(result), nil
	}

	element, err := schema.UnmarshalMap(convert.MapOfAny(data))

	if err != nil {
		return schema.Schema{}, derp.Wrap(err, location, "Unmarshalling schema")
	}

	// RULE: The schema describes a JSON object
	object, isObject := element.(schema.Object)

	if !isObject {
		return schema.Schema{}, derp.Validation("Schema must be an object")
	}

	// RULE: Every list has a limit, because rosetta bounds lists only through the schema
	if path, found := arrayWithoutMaxLength(object, ""); found {
		return schema.Schema{}, derp.Validation("Array must have a maxLength: " + path)
	}

	// The Template's own properties win, and any other property stays allowed
	for name, property := range object.Properties {
		result.Properties[name] = property
	}

	if object.Wildcard != nil {
		result.Wildcard = object.Wildcard
	}

	return schema.New(result), nil
}

// arrayWithoutMaxLength returns the path of the first Array beneath the element that has no MaxLength
func arrayWithoutMaxLength(element schema.Element, path string) (string, bool) {

	switch typed := element.(type) {

	case schema.Array:

		if typed.MaxLength <= 0 {
			return path, true
		}

		return arrayWithoutMaxLength(typed.Items, path+".*")

	case schema.Object:

		for name, property := range typed.Properties {
			if found, ok := arrayWithoutMaxLength(property, joinPath(path, name)); ok {
				return found, true
			}
		}

		if typed.Wildcard != nil {
			return arrayWithoutMaxLength(typed.Wildcard, joinPath(path, "*"))
		}
	}

	return "", false
}

// joinPath appends a name to a dotted path
func joinPath(path string, name string) string {

	if path == "" {
		return name
	}

	return path + "." + name
}

// Name returns the name of the step, which is used in debugging.
func (step ViewJSON) Name() string {
	return "view-json"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ViewJSON) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewJSON) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewJSON) RequiredRoles() []string {
	return []string{}
}
