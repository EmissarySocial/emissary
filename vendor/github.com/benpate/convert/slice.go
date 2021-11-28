package convert

import "strconv"

// SliceOfString converts the value into a slice of strings.
// It works with interface{}, []interface{}, []string, and string values.
// If the passed value cannot be converted, then an empty slice is returned.
func SliceOfString(value interface{}) []string {

	switch value := value.(type) {

	case string:
		return []string{value}

	case []string:
		return value

	case []int:
		result := make([]string, len(value))
		for index, v := range value {
			result[index] = strconv.Itoa(v)
		}
		return result

	case []float64:
		result := make([]string, len(value))
		for index, v := range value {
			result[index] = String(v)
		}
		return result

	case []interface{}:
		result := make([]string, len(value))
		for index, v := range value {
			result[index] = String(v)
		}
		return result
	}

	return make([]string, 0)
}

// SliceOfInt converts the value into a slice of ints.
// It works with interface{}, []interface{}, []int, and int values.
// If the passed value cannot be converted, then an empty slice is returned.
func SliceOfInt(value interface{}) []int {

	switch value := value.(type) {

	case []interface{}:
		result := make([]int, len(value))
		for index, v := range value {
			result[index] = Int(v)
		}
		return result

	case []int:
		return value

	case int:
		return []int{value}
	}

	return make([]int, 0)
}

// SliceOfFloat converts the value into a slice of floats.
// It works with interface{}, []interface{}, []float64, and float64 values.
// If the passed value cannot be converted, then an empty slice is returned.
func SliceOfFloat(value interface{}) []float64 {

	switch value := value.(type) {

	case []interface{}:
		result := make([]float64, len(value))
		for index, v := range value {
			result[index] = Float(v)
		}
		return result

	case []float64:
		return value

	case float64:
		return []float64{value}
	}

	return make([]float64, 0)
}

// SliceOfMap converts the value into a slice of map[string]interface{}.
// It works with []interface{}, []map[string]interface{}.
// If the passed value cannot be converted, then an empty slice is returned.
func SliceOfMap(value interface{}) []map[string]interface{} {

	switch value := value.(type) {

	case []map[string]interface{}:
		return value

	case []interface{}:
		result := make([]map[string]interface{}, len(value))
		for index, v := range value {

			if mapValue, ok := v.(map[string]interface{}); ok {
				result[index] = mapValue
			} else {
				result[index] = map[string]interface{}{}
			}
		}
		return result
	}

	return make([]map[string]interface{}, 0)
}
