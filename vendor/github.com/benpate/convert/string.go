package convert

import (
	"strconv"
)

// String forces a conversion from an arbitrary value into an string.
// If the value cannot be converted, then the default value for the type is used.
func String(value interface{}) string {

	result, _ := StringOk(value, "")
	return result
}

// StringDefault forces a conversion from an arbitrary value into a string.
// if the value cannot be converted, then the default value is used.
func StringDefault(value interface{}, defaultValue string) string {

	result, _ := StringOk(value, defaultValue)
	return result
}

// StringOk converts an arbitrary value (passed in the first parameter) into a string, no matter what.
// The first result is the final converted value, or the default value (passed in the second parameter)
// The second result is TRUE if the value was naturally a string, and FALSE otherwise
//
// Conversion Rules:
// Nils return default value and Ok=false
// Bools are formated as "true" or "false" with Ok=false
// Ints are formated as strings with Ok=false
// Floats are formatted with 2 decimal places, with Ok=false
// String are passed through directly, with Ok=true
// Known interfaces (Inter, Floater, Stringer) are handled like their corresponding types.
// All other values return the default value with Ok=false
func StringOk(value interface{}, defaultValue string) (string, bool) {

	if value == nil {
		return defaultValue, false
	}

	switch v := value.(type) {

	case bool:

		if v {
			return "true", false
		}

		return "false", false

	case []byte:
		return string(v), true

	case int:
		return strconv.Itoa(v), false

	case int8:
		return strconv.FormatInt(int64(v), 10), false

	case int16:
		return strconv.FormatInt(int64(v), 10), false

	case int32:
		return strconv.FormatInt(int64(v), 10), false

	case int64:
		return strconv.FormatInt(v, 10), false

	case float32:
		return strconv.FormatFloat(float64(v), 'f', -2, 64), false

	case float64:
		return strconv.FormatFloat(v, 'f', -2, 64), false

	case string:
		return v, true

	case Booler:
		return StringOk(v.Bool(), defaultValue)

	case Inter:
		return strconv.FormatInt(int64(v.Int()), 10), false

	case Floater:
		return strconv.FormatFloat(v.Float(), 'f', -2, 64), false

	case Stringer:
		return v.String(), true
	}

	return defaultValue, false
}
