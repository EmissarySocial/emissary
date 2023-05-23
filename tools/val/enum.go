package val

// Enum verifies that the value exists in the list of provided values.
// If the value exists, then it is returned.
// If it does not, then the first item in the list is returned
// If the list is empty, then the zero value for that type is returned.
func Enum[T comparable](value T, enum ...T) T {

	if len(enum) == 0 {
		var zero T
		return zero
	}

	for _, e := range enum {
		if value == e {
			return value
		}
	}

	return enum[0]
}
