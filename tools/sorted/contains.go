package sorted

import (
	"strings"
)

// Contains is an efficient way to search a sorted slice for a value.
// IMPORTANT: The slice must be sorted in order for this function to work.
// If the slice is unsorted, the result will be inaccurate.
func Contains(values []string, target string) bool {

	for _, value := range values {

		switch strings.Compare(value, target) {

		case -1:

		case 0:
			return true

		case 1:
			return false
		}
	}

	return false
}

// ContainsAll is an efficient way to compare two sorted slices
// to see if one is completely contained within the other.
// IMPORTANT: Both slices must be sorted in order for this function to work.
// If either slice is unsorted, the result will be inaccurate.
func ContainsAll(subset []string, superset []string) bool {

	subsetIndex := 0
	subsetLength := len(subset)

	supersetIndex := 0
	supersetLength := len(superset)

	for {

		// If we have successfully scanned the whole subset,
		// then we know that the subset is contained within the superset.
		if subsetIndex >= subsetLength {
			return true
		}

		// If we have overflowed the superset and are still searching,
		// then we know that the subset is NOT contained within the superset.
		if supersetIndex >= supersetLength {
			return false
		}

		// Compare the next two items
		switch strings.Compare(subset[subsetIndex], superset[supersetIndex]) {

		case -1:
			return false

		case 0:
			subsetIndex++
			supersetIndex++

		case 1:
			supersetIndex++
		}
	}
}
