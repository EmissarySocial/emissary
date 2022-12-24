package slices

import "strconv"

type Slice[T any] []T

func (slice *Slice[T]) GetChild(path string) (any, bool) {

	// Try to get the path as an integer
	index, err := strconv.Atoi(path)

	if err != nil {
		return nil, false
	}

	// Bounds checking
	if index < 0 {
		return nil, false
	}

	if index >= len(*slice) {
		return nil, false // TODO: LOW: Could we add a new item to the slice?
	}

	// Return reference to the value.
	return &(*slice)[index], true
}
