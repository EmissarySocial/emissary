package derp

// ErrorCodeGetter interface describes any error that can also "get" an error code value
type ErrorCodeGetter interface {

	// ErrorCode returns a numeric, application-specific code that references this error.
	// HTTP status codes are recommended, but not required
	ErrorCode() int
}

// ErrorCodeSetter interface describes any error that can also "set" an error code value
type ErrorCodeSetter interface {

	// SetErrorCode sets a numeric, application-specific code that for this error.
	// HTTP status codes are recommended, but not required
	SetErrorCode(int)
}

// MessageGetter interface describes any error that can also report a "Message"
type MessageGetter interface {

	// Message returns a human-friendly string representation of the error.
	Message() string
}

// Unwrapper interface describes any error that can be "unwrapped".  It supports
// the Unwrap method added in Go 1.13+
type Unwrapper interface {

	// Unwrap returns the inner error bundled inside of an outer error.
	Unwrap() error
}
