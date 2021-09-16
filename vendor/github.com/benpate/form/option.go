package form

// OptionProvider is an external object that
// can inject OptionCodes based on their URL.
type OptionProvider interface {
	OptionCodes(string) ([]OptionCode, error)
}

// OptionCode represents a single value/label pair
// to be used in place of Enums for optional lists.
type OptionCode struct {
	Label string
	Value string
}
