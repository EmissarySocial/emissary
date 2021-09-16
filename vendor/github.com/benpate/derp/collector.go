package derp

// Collector collects multiple errors into a single MultiError.
// A Collector is not an error value itself, but is able to
// generate an error at the end of a function.
type Collector struct {
	multiError MultiError
}

// NewCollector generates a fully populated Collector object.
func NewCollector() *Collector {
	return &Collector{
		multiError: make([]error, 0),
	}
}

// Add appends one or more errors into the collected MultiError.
// If a nil value is passed in, then no operation is taken.
// If another MultiError is passed in, then its sub-values are flattened
// into this value.
func (c *Collector) Add(errs ...error) {

	for _, nextError := range errs {

		if isNil(nextError) {
			continue
		}

		if nextMultiError, ok := nextError.(MultiError); ok {
			c.multiError = append(c.multiError, nextMultiError...)
			continue
		}

		c.multiError = append(c.multiError, nextError)
	}

}

// Error returns the error value of the collected MultiError.
func (c *Collector) Error() error {

	if len(c.multiError) == 0 {
		return nil
	}

	return c.multiError
}
