package compare

// Int compares two int values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Int(value1 int, value2 int) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// Int8 compares two int8 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Int8(value1 int8, value2 int8) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// Int16 compares two int16 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Int16(value1 int16, value2 int16) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// Int32 compares two int32 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Int32(value1 int32, value2 int32) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// Int64 compares two int64 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func Int64(value1 int64, value2 int64) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// UInt compares two uint values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func UInt(value1 uint, value2 uint) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// UInt8 compares two uint8 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func UInt8(value1 uint8, value2 uint8) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// UInt16 compares two uint16 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func UInt16(value1 uint16, value2 uint16) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// UInt32 compares two uint32 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func UInt32(value1 uint32, value2 uint32) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}

// UInt64 compares two uint64 values.
// It returns -1 if value1 is LESS THAN value2.
// It returns 0 if value1 is EQUAL TO value2.
// It returns 1 if value1 is GREATER THAN value2.
func UInt64(value1 uint64, value2 uint64) int {

	switch {

	case value1 > value2:
		return 1
	case value1 == value2:
		return 0
	default:
		return -1
	}
}
