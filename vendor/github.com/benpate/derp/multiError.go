package derp

import "strings"

// MultiError represents a runtime error.  It includes
type MultiError []error

// Message retrieves the error message from the first message in the slice that is a messageGetter
func (m MultiError) Message() string {

	if len(m) == 0 {
		return ""
	}

	for _, err := range m {

		if message := Message(err); message != "" {
			return message
		}
	}

	return "Unrecognized Error"
}

// Error implements the Error interface, which allows derp.Error objects to be
// used anywhere a standard error is used.
func (m MultiError) Error() string {

	b := strings.Builder{}

	for _, err := range m {
		b.WriteString(err.Error() + "\n")
	}

	return b.String()
}

// ErrorCode returns the error Code embedded in this Error.  This is useful for matching
// interfaces in other package.
func (m MultiError) ErrorCode() int {

	if len(m) == 0 {
		return 0
	}

	for _, err := range m {

		if errorWithCode, ok := err.(ErrorCodeGetter); ok {
			return errorWithCode.ErrorCode()
		}
	}

	return 500
}
