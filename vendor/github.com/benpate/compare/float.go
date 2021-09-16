package compare

// Float32 compares two float32 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Float32(value1 float32, value2 float32) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// Float64 compares two float64 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Float64(value1 float64, value2 float64) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}
