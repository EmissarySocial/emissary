package stripe

// nolint:unused - It's okay if this is unused from time to time.
func iif[T any](condition bool, trueValue T, falseValue T) T {
	if condition {
		return trueValue
	}
	return falseValue
}
