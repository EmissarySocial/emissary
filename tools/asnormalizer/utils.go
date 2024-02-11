package asnormalizer

import "github.com/benpate/hannibal/streams"

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

// biggestImage scans a (possible) array of images and returns the
// value that has the largest width.
func biggestImage(document streams.Document) streams.Document {

	max := document.Head()

	for document = document.Tail(); document.NotNil(); document = document.Tail() {
		if document.Width() > max.Width() {
			max = document.Head()
		}
	}

	return max
}
