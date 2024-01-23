package asnormalizer

// first returns the first non-zero value from a list of values
func first[T comparable](values ...T) T {
	var zero T
	for _, value := range values {
		if value != zero {
			return value
		}
	}
	return zero
}
