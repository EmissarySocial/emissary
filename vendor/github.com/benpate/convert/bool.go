package convert

// Bool forces a conversion from an arbitrary value into a boolean.
// If the value cannot be converted, then the default value for the type is used.
func Bool(value interface{}) bool {

	result, _ := BoolOk(value, false)
	return result
}

// BoolDefault forces a conversion from an arbitrary value into a bool.
// if the value cannot be converted, then the default value is used.
func BoolDefault(value interface{}, defaultValue bool) bool {

	result, _ := BoolOk(value, defaultValue)
	return result
}

// BoolOk converts an arbitrary value (passed in the first parameter) into a boolean, somehow, no matter what.
// The first result is the final converted value, or the default value (passed in the second parameter)
// The second result is TRUE if the value was naturally a bool, and FALSE otherwise
//
// Conversion Rules:
// Nils return default value and Ok=false
// Bools are passed through with Ok=true
// Ints and Floats all convert to FALSE if they are zero, and TRUE if they are non-zero.  In these cases, Ok=false
// String values of "true" and "false" convert normall, and Ok=true.  All other strings return the default value, with Ok=false
// Known interfaces (Booler, Inter, Floater, Stringer) are handled like their corresponding types
// All other values return the default value with Ok=false
func BoolOk(value interface{}, defaultValue bool) (bool, bool) {

	if value == nil {
		return defaultValue, false
	}

	switch v := value.(type) {

	case bool:
		return v, true

	case int:
		return (v != 0), false

	case int8:
		return (v != 0), false

	case int16:
		return (v != 0), false

	case int32:
		return (v != 0), false

	case int64:
		return (v != 0), false

	case float32:
		return (v != 0), false

	case float64:
		return (v != 0), false

	case string:

		switch v {
		case "true":
			return true, true
		case "false":
			return false, true
		default:
			return defaultValue, false
		}

	case Booler:
		return BoolOk(v.Bool(), defaultValue)

	case Inter:
		return BoolOk(v.Int(), defaultValue)

	case Floater:
		return BoolOk(v.Float(), defaultValue)

	case Stringer:
		return BoolOk(v.String(), defaultValue)
	}

	return defaultValue, false
}
