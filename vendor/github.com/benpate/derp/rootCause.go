package derp

// RootCause digs into the error stack and returns the original error
// that caused the DERP
func RootCause(err error) error {

	// If this error can be "unwrapped" then dig deeper into the chain
	if unwrapper, ok := err.(Unwrapper); ok {

		// Try to unwrap the error.  If it is a not-Nil result, then keep digging
		if next := unwrapper.Unwrap(); !isNil(next) {
			return RootCause(next)
		}
	}

	// Fall through means that there is nothing left to unwrap.  Return the current error
	return err
}
