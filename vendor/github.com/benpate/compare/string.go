package compare

import "strings"

// String compares two string values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func String(value1 string, value2 string) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// BeginsWith is a simple "generic-safe" function for string comparison.  It returns TRUE if value1 begins with value2
func BeginsWith(value1 interface{}, value2 interface{}) bool {

	if value1, ok := value1.(string); ok {

		if value2, ok := value2.(string); ok {

			return strings.HasPrefix(value1, value2)
		}
		return false
	}

	if value1, ok := value1.([]string); ok {
		if len(value1) > 0 {
			if value2, ok := value2.(string); ok {
				return (value1[0] == value2)
			}
		}
		return false
	}

	return false
}

// EndsWith is a simple "generic-safe" function for string comparison.  It returns TRUE if value1 ends with value2
func EndsWith(value1 interface{}, value2 interface{}) bool {

	if value1, ok := value1.(string); ok {

		if value2, ok := value2.(string); ok {
			return strings.HasSuffix(value1, value2)
		}
		return false
	}

	if value1, ok := value1.([]string); ok {
		if len(value1) > 0 {
			if value2, ok := value2.(string); ok {
				return (value1[len(value1)-1] == value2)
			}
		}
		return false
	}

	return false
}
