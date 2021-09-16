package derp

// SingleError represents a runtime error.  It includes
type SingleError struct {
	Code       int           `json:"code"`       // Numeric error code (such as an HTTP status code) to report to the client.
	Location   string        `json:"location"`   // Function name (or other location description) of where the error occurred
	Message    string        `json:"message"`    // Primary (top-level) error message for this error
	TimeStamp  int64         `json:"timestamp"`  // Unix Epoch timestamp of the date/time when this error was created
	Details    []interface{} `json:"details"`    // Additional information related to this error message, such as parameters to the function that caused the error.
	InnerError error         `json:"innerError"` // An underlying error object used to identify the root cause of this error.
}

// Error implements the Error interface, which allows derp.Error objects to be
// used anywhere a standard error is used.
func (err SingleError) Error() string {
	return err.Location + ": " + err.Message
}

// ErrorCode returns the error Code embedded in this Error.  This is useful for matching
// interfaces in other package.
func (err SingleError) ErrorCode() int {
	return err.Code
}

// SetErrorCode returns the error Code embedded in this Error.  This is useful for matching
// interfaces in other package.
func (err *SingleError) SetErrorCode(code int) {
	err.Code = code
}

// Unwrap supports Go 1.13+ error unwrapping
func (err SingleError) Unwrap() error {
	return err.InnerError
}
