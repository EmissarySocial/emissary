package derp

// Message retrieves the best-fit error message for any type of error
func Message(err error) string {

	if isNil(err) {
		return ""
	}

	switch e := err.(type) {
	case *SingleError:
		return e.Message

	case MessageGetter:
		return e.Message()
	}

	return err.Error()
}

// NotFound returns TRUE if the error `Code` is a 404 / Not Found error.
func NotFound(err error) bool {

	if isNil(err) {
		return false
	}

	if coder, ok := err.(ErrorCodeGetter); ok {
		return coder.ErrorCode() == CodeNotFoundError
	}

	return err.Error() == "not found"
}

// ErrorCode returns an error code for any error.  It tries to read the error code
// from objects matching the ErrorCodeGetter interface.  If the provided error does not
// match this interface, then it assigns a generic "Internal Server Error" code 500.
func ErrorCode(err error) int {

	if isNil(err) {
		return 0
	}

	if getter, ok := err.(ErrorCodeGetter); ok {
		return getter.ErrorCode()
	}

	return CodeInternalError
}

// SetErrorCode tries to set an error code for the provided error.  If the error matches the
// ErrorCodeSetter interface, then the code is set directly in the error.  Otherwise,
// it has no effect.
func SetErrorCode(err error, code int) {

	if isNil(err) {
		return
	}

	if setter, ok := err.(ErrorCodeSetter); ok {
		setter.SetErrorCode(code)
	}
}
