package schema

import (
	"strings"

	"github.com/benpate/compare"
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/list"
	"github.com/benpate/null"
	"github.com/benpate/path"
)

// String represents a string data type within a JSON-Schema.
type String struct {
	Default   string
	MinLength null.Int
	MaxLength null.Int
	Enum      []string
	Pattern   string
	Format    string
	Required  bool
}

// Enumerate implements the "Enumerator" interface
func (str String) Enumerate() []string {
	return str.Enum
}

// Type returns the data type of this Element
func (str String) Type() Type {
	return TypeString
}

// Path returns sub-schemas or an error
func (str String) Path(p path.Path) (Element, error) {

	if p.IsEmpty() {
		return str, nil
	}

	return nil, derp.New(500, "schema.String.GetPath", "String values have no child elements.  Path must terminate.", p)
}

// Validate compares a generic data value using this Schema
func (str String) Validate(value interface{}) error {

	// Try to convert the value to a string
	stringValue, ok := value.(string)

	// Fail if not a string
	if !ok {
		return ValidationError{Message: "must be a string"}
	}

	if str.Required {
		if stringValue == "" {
			return ValidationError{Message: "field is required"}
		}
	}

	result := derp.NewCollector()

	if str.MinLength.IsPresent() {
		if len(stringValue) < str.MinLength.Int() {
			result.Add(ValidationError{Message: "minimum length is " + str.MinLength.String()})
		}
	}

	if str.MaxLength.IsPresent() {
		if len(stringValue) > str.MaxLength.Int() {
			result.Add(ValidationError{Message: "Maximum length is " + str.MaxLength.String()})
		}
	}

	if len(str.Enum) > 0 {
		if !compare.Contains(str.Enum, stringValue) {
			result.Add(ValidationError{Message: "must match one of the required values."})
		}
	}

	if str.Format != "" {

		formatParams := strings.Split(str.Format, " ")

		for _, arg := range formatParams {

			name, arg := list.Split(arg, "=")

			if fn, ok := formats[name]; ok {
				if err := fn(arg)(stringValue); err != nil {
					result.Add(err)
				}
			}
		}
	}

	if str.Pattern != "" {
		// TODO: check custom patterns...
	}

	return result.Error()
}

// MarshalMap populates object data into a map[string]interface{}
func (str String) MarshalMap() map[string]interface{} {

	result := map[string]interface{}{
		"type":     str.Type(),
		"required": str.Required,
	}

	if str.Default != "" {
		result["default"] = str.Default
	}

	if str.MinLength.IsPresent() {
		result["minLength"] = str.MinLength.Int()
	}

	if str.MaxLength.IsPresent() {
		result["maxLength"] = str.MaxLength.Int()
	}

	if str.Pattern != "" {
		result["pattern"] = str.Pattern
	}

	if str.Format != "" {
		result["format"] = str.Format
	}

	if len(str.Enum) > 0 {
		result["enum"] = str.Enum
	}

	return result
}

// UnmarshalMap tries to populate this object using data from a map[string]interface{}
func (str *String) UnmarshalMap(data map[string]interface{}) error {

	var err error

	if convert.String(data["type"]) != "string" {
		return derp.New(500, "schema.String.UnmarshalMap", "Data is not type 'string'", data)
	}

	str.Default = convert.String(data["default"])
	str.MinLength = convert.NullInt(data["minLength"])
	str.MaxLength = convert.NullInt(data["maxLength"])
	str.Pattern = convert.String(data["pattern"])
	str.Format = convert.String(data["format"])
	str.Required = convert.Bool(data["required"])
	str.Enum = convert.SliceOfString(data["enum"])

	return err
}

func (str String) MarshalJavascript(b *strings.Builder) {

	if str.Required {
		b.WriteString(` if (v=="") {return false;}`)
	}

	if len(str.Enum) > 0 {
		b.WriteString("")
	}

	b.WriteString(`return true;`)
}
