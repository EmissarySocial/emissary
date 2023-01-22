package step

func getValue[T any](value T, _ bool) T {
	return value
}
