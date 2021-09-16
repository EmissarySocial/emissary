package schema

import (
	"strings"

	"github.com/benpate/compare"
	"github.com/benpate/convert"
	"github.com/benpate/derp"
	"github.com/benpate/null"
	"github.com/benpate/path"
)

// Number represents a number data type within a JSON-Schema.
type Number struct {
	Default  null.Float `json:"default"`
	Minimum  null.Float `json:"minimum"`
	Maximum  null.Float `json:"maximum"`
	Enum     []float64  `json:"enum"`
	Required bool
}

// Enumerate implements the "Enumerator" interface
func (number Number) Enumerate() []string {
	return convert.SliceOfString(number.Enum)
}

// Type returns the data type of this Element
func (number Number) Type() Type {
	return TypeNumber
}

// Path returns sub-schemas
func (number Number) Path(p path.Path) (Element, error) {

	if p.IsEmpty() {
		return number, nil
	}

	return nil, derp.New(500, "schema.Number.GetPath", "Number values have no child elements.  Path must terminate.", p)
}

// Validate compares a generic data value using this Schema
func (number Number) Validate(value interface{}) error {

	// Try to convert the value to a string
	numberValue, numberValueOk := convert.FloatOk(value, 0)

	// Fail if not a number
	if !numberValueOk {
		return ValidationError{Message: "must be a number"}
	}

	if number.Required {
		if numberValue == 0 {
			return ValidationError{Message: "field is required"}
		}
	}

	result := derp.NewCollector()

	if number.Minimum.IsPresent() {
		if numberValue <= number.Minimum.Float() {
			result.Add(ValidationError{Message: "minimum value is" + convert.String(number.Minimum)})
		}
	}

	if number.Maximum.IsPresent() {
		if numberValue >= number.Maximum.Float() {
			result.Add(ValidationError{Message: "maximum value is " + convert.String(number.Maximum)})
		}
	}

	if len(number.Enum) > 0 {
		if !compare.Contains(number.Enum, numberValue) {
			result.Add(ValidationError{Message: "must contain one of the specified values"})
		}
	}

	return result.Error()
}

// MarshalMap populates object data into a map[string]interface{}
func (number Number) MarshalMap() map[string]interface{} {

	result := map[string]interface{}{
		"type": number.Type(),
	}

	if number.Default.IsPresent() {
		result["default"] = number.Default.Float()
	}

	if number.Minimum.IsPresent() {
		result["minimum"] = number.Minimum.Float()
	}

	if number.Maximum.IsPresent() {
		result["maximum"] = number.Maximum.Float()
	}

	if len(number.Enum) > 0 {
		result["enum"] = number.Enum
	}

	return result
}

// UnmarshalMap tries to populate this object using data from a map[string]interface{}
func (number *Number) UnmarshalMap(data map[string]interface{}) error {

	var err error

	if convert.String(data["type"]) != "number" {
		return derp.New(500, "schema.Number.UnmarshalMap", "Data is not type 'number'", data)
	}

	number.Default = convert.NullFloat(data["default"])
	number.Minimum = convert.NullFloat(data["minimum"])
	number.Maximum = convert.NullFloat(data["maximum"])
	number.Required = convert.Bool(data["required"])
	number.Enum = convert.SliceOfFloat(data["enum"])

	return err
}

func (number Number) MarshalJavascript(b *strings.Builder) {

	//	b.WriteString(`v = Math.parseFloat(v)`)
}
