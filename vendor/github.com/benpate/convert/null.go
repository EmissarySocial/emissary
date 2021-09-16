package convert

import "github.com/benpate/null"

// NullBool converts a value into a nullable value.
// The value is only set if the input value is a natural match for this data type.
func NullBool(value interface{}) null.Bool {

	var result null.Bool

	if v, ok := BoolOk(value, false); ok {
		result.Set(v)
	}

	return result
}

// NullInt converts a value into a nullable value.
// The value is only set if the input value is a natural match for this data type.
func NullInt(value interface{}) null.Int {

	var result null.Int

	if v, ok := IntOk(value, 0); ok {
		result.Set(v)
	}

	return result

}

// NullFloat converts a value into a nullable value.
// The value is only set if the input value is a natural match for this data type.
func NullFloat(value interface{}) null.Float {

	var result null.Float

	if v, ok := FloatOk(value, 0); ok {
		result.Set(v)
	}

	return result

}
