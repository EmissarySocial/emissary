package convert

import (
	"math"
	"strconv"
)

// Int forces a conversion from an arbitrary value into an int.
// If the value cannot be converted, then the zero value for the type (0) is used.
func Int(value interface{}) int {

	result, _ := IntOk(value, 0)
	return result
}

// IntDefault forces a conversion from an arbitrary value into a int.
// if the value cannot be converted, then the default value is used.
func IntDefault(value interface{}, defaultValue int) int {

	result, _ := IntOk(value, defaultValue)
	return result
}

// IntOk converts an arbitrary value (passed in the first parameter) into an int, no matter what.
// The first result is the final converted value, or the default value (passed in the second parameter)
// The second result is TRUE if the value was naturally an integer, and FALSE otherwise
//
// Conversion Rules:
// Nils and Bools return default value and Ok=false
// Ints are returned directly with Ok=true
// Floats are truncated into ints.  If there is no decimal value then Ok=true
// String values are attempted to parse as a int.  If unsuccessful, default value is returned.  For all strings, Ok=false
// Known interfaces (Inter, Floater, Stringer) are handled like their corresponding types.
// All other values return the default value with Ok=false
func IntOk(value interface{}, defaultValue int) (int, bool) {

	if value == nil {
		return defaultValue, false
	}

	switch v := value.(type) {

	case int:
		return int(v), true

	case int8:
		return int(v), true

	case int16:
		return int(v), true

	case int32:
		return int(v), true

	case int64:
		return int(v), true

	case float32:
		return int(v), hasDecimal(float64(v))

	case float64:
		return int(v), hasDecimal(v)

	case string:
		result, err := strconv.Atoi(v)

		if err != nil {
			return defaultValue, false
		}

		return result, false

	case Inter:
		return v.Int(), true

	case Floater:
		result := v.Float()
		return int(result), hasDecimal(result)

	case Stringer:
		return IntOk(v.String(), defaultValue)

	}

	return defaultValue, false
}

func hasDecimal(value float64) bool {

	return (value == math.Floor(value))
}
