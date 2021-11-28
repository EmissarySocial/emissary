package model

// Option is a general-purpose value used to pass around multi-select setting values.
type Option struct {
	Value       string // Internal value of the Option
	Label       string // Human-friendly label/name of the Option
	Description string // Optional long description of the Option
	IconURL     string // Optional icon to use when displaying the Option
	Group       string // Optiional grouping to use when displaying the Option
}
