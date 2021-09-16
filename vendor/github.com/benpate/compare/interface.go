package compare

import "github.com/benpate/derp"

// Interface tries its best to muscle value2 and value2 into compatable types so that they can be compared.
// If value1 is LESS THAN value2, it returns -1, nil
// If value1 is EQUAL TO value2, it returns 0, nil
// If value1 is GREATER THAN value2, it returns 1, nil
// If the two values are not compatable, then it returns 0, [DERP] with an explanation of the error.
// Currently, this function ONLLY compares identical numeric or string types.  In the future, it *may*
// be expanded to perform simple type converstions between similar types.
func Interface(value1 interface{}, value2 interface{}) (int, error) {

	switch v1 := value1.(type) {

	case int:

		switch v2 := value2.(type) {

		case int:
			return Int64(int64(v1), int64(v2)), nil

		case int8:
			return Int64(int64(v1), int64(v2)), nil

		case int16:
			return Int64(int64(v1), int64(v2)), nil

		case int32:
			return Int64(int64(v1), int64(v2)), nil

		case int64:
			return Int64(int64(v1), int64(v2)), nil

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case int8:

		switch v2 := value2.(type) {

		case int:
			return Int64(int64(v1), int64(v2)), nil

		case int8:
			return Int64(int64(v1), int64(v2)), nil

		case int16:
			return Int64(int64(v1), int64(v2)), nil

		case int32:
			return Int64(int64(v1), int64(v2)), nil

		case int64:
			return Int64(int64(v1), int64(v2)), nil

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case int16:

		switch v2 := value2.(type) {

		case int:
			return Int64(int64(v1), int64(v2)), nil

		case int8:
			return Int64(int64(v1), int64(v2)), nil

		case int16:
			return Int64(int64(v1), int64(v2)), nil

		case int32:
			return Int64(int64(v1), int64(v2)), nil

		case int64:
			return Int64(int64(v1), int64(v2)), nil

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case int32:

		switch v2 := value2.(type) {

		case int:
			return Int64(int64(v1), int64(v2)), nil

		case int8:
			return Int64(int64(v1), int64(v2)), nil

		case int16:
			return Int64(int64(v1), int64(v2)), nil

		case int32:
			return Int64(int64(v1), int64(v2)), nil

		case int64:
			return Int64(int64(v1), int64(v2)), nil

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case int64:

		switch v2 := value2.(type) {

		case int:
			return Int64(int64(v1), int64(v2)), nil

		case int8:
			return Int64(int64(v1), int64(v2)), nil

		case int16:
			return Int64(int64(v1), int64(v2)), nil

		case int32:
			return Int64(int64(v1), int64(v2)), nil

		case int64:
			return Int64(int64(v1), int64(v2)), nil

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case uint:

		switch v2 := value2.(type) {

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint8:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint16:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint32:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint64:
			return UInt64(uint64(v1), v2), nil // TODO: range checking?

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case uint8:

		switch v2 := value2.(type) {

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint8:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint16:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint32:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint64:
			return UInt64(uint64(v1), v2), nil // TODO: range checking?

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case uint16:

		switch v2 := value2.(type) {

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint8:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint16:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint32:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint64:
			return UInt64(uint64(v1), v2), nil // TODO: range checking?

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case uint32:

		switch v2 := value2.(type) {

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint8:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint16:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint32:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint64:
			return UInt64(uint64(v1), v2), nil // TODO: range checking?

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case uint64:

		switch v2 := value2.(type) {

		case uint:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint8:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint16:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint32:
			return UInt64(uint64(v1), uint64(v2)), nil

		case uint64:
			return UInt64(uint64(v1), v2), nil // TODO: range checking?

		case float32:
			return Float32(float32(v1), v2), nil

		case float64:
			return Float64(float64(v1), v2), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case float32:

		switch v2 := value2.(type) {

		case int:
			return Float64(float64(v1), float64(v2)), nil

		case int8:
			return Float64(float64(v1), float64(v2)), nil

		case int16:
			return Float64(float64(v1), float64(v2)), nil

		case int32:
			return Float64(float64(v1), float64(v2)), nil

		case int64:
			return Float64(float64(v1), float64(v2)), nil

		case uint:
			return Float64(float64(v1), float64(v2)), nil

		case uint8:
			return Float64(float64(v1), float64(v2)), nil

		case uint16:
			return Float64(float64(v1), float64(v2)), nil

		case uint32:
			return Float64(float64(v1), float64(v2)), nil

		case uint64:
			return Float64(float64(v1), float64(v2)), nil

		case float32:
			return Float64(float64(v1), float64(v2)), nil

		case float64:
			return Float64(float64(v1), float64(v2)), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case float64:

		switch v2 := value2.(type) {

		case int:
			return Float64(float64(v1), float64(v2)), nil

		case int8:
			return Float64(float64(v1), float64(v2)), nil

		case int16:
			return Float64(float64(v1), float64(v2)), nil

		case int32:
			return Float64(float64(v1), float64(v2)), nil

		case int64:
			return Float64(float64(v1), float64(v2)), nil

		case uint:
			return Float64(float64(v1), float64(v2)), nil

		case uint8:
			return Float64(float64(v1), float64(v2)), nil

		case uint16:
			return Float64(float64(v1), float64(v2)), nil

		case uint32:
			return Float64(float64(v1), float64(v2)), nil

		case uint64:
			return Float64(float64(v1), float64(v2)), nil

		case float32:
			return Float64(float64(v1), float64(v2)), nil

		case float64:
			return Float64(float64(v1), float64(v2)), nil

		default:
			return 0, derp.New(500, "compare.Interface", "Incompatible data type", value1, value2)
		}

	case string:

		if v2, ok := value2.(string); ok {
			return String(v1, v2), nil
		}

	case Stringer:

		if v2, ok := value2.(Stringer); ok {
			return String(v1.String(), v2.String()), nil
		}
	}

	return 0, derp.New(500, "compare.Interface", "Incompatible Types", value1, value2)
}

// Stringer is an interface for types that can be converted to String
type Stringer interface {
	String() string
}
