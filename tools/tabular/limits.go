package tabular

// Limits bounds what Parse reads from one file.  A zero or negative value means the default.
type Limits struct {
	MaxBytes     int // Largest file, in bytes
	MaxRows      int // Most rows, not counting a header
	MaxColumns   int // Most cells in one row
	MaxCellBytes int // Longest text value in one cell, in bytes
	MaxDepth     int // Deepest nesting of objects and arrays within one row
	MaxProblems  int // Most Problems reported before Parse stops reading
}

// DefaultLimits returns the Limits that Parse uses when none are set
func DefaultLimits() Limits {
	return Limits{
		MaxBytes:     DefaultMaxBytes,
		MaxRows:      DefaultMaxRows,
		MaxColumns:   DefaultMaxColumns,
		MaxCellBytes: DefaultMaxCellBytes,
		MaxDepth:     DefaultMaxDepth,
		MaxProblems:  DefaultMaxProblems,
	}
}

// withDefaults returns a copy of these Limits with every unset value replaced by its default
func (limits Limits) withDefaults() Limits {
	defaults := DefaultLimits()

	limits.MaxBytes = positiveOr(limits.MaxBytes, defaults.MaxBytes)
	limits.MaxRows = positiveOr(limits.MaxRows, defaults.MaxRows)
	limits.MaxColumns = positiveOr(limits.MaxColumns, defaults.MaxColumns)
	limits.MaxCellBytes = positiveOr(limits.MaxCellBytes, defaults.MaxCellBytes)
	limits.MaxDepth = positiveOr(limits.MaxDepth, defaults.MaxDepth)
	limits.MaxProblems = positiveOr(limits.MaxProblems, defaults.MaxProblems)

	return limits
}

// positiveOr returns value when it is positive, and fallback otherwise
func positiveOr(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}
