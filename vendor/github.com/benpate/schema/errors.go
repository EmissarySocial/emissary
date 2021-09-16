package schema

import (
	"github.com/benpate/derp"
	"github.com/benpate/list"
)

// ValidationErrorCode represents HTTP Status Code: 422 "Unproccessable Entity"
const ValidationErrorCode = 422

// ValidationError represents an input validation error, and includes fields necessary to
// report problems back to the end user.
type ValidationError struct {
	Path    string `json:"path"`    // Identifies the PATH (or variable name) that has invalid input
	Message string `json:"message"` // Human-readable message that explains the problem with the input value.
}

// Invalid returns a fully populated ValidationError to the caller
func Invalid(message string) ValidationError {
	return ValidationError{
		Path:    "",
		Message: message,
	}
}

// Error returns a string representation of this ValidationError, and implements
// the builtin errors.error interface.
func (v ValidationError) Error() string {
	return v.Message
}

// ErrorCode returns CodeValidationError for this ValidationError
// It implements the ErrorCodeGetter interface.
func (v ValidationError) ErrorCode() int {
	return ValidationErrorCode
}

// Rollup bundles a child error into a parent
func Rollup(errs error, path string) error {

	if errs == nil {
		return nil
	}

	if multiError, ok := errs.(*derp.MultiError); ok {

		for index := range *multiError {

			// If the child error is nil for some reason, then skip this record.
			if (*multiError)[index] == nil {
				continue
			}

			if validationError, ok := (*multiError)[index].(ValidationError); ok {
				validationError.Path = list.PushHead(validationError.Path, path, ".")
			}
		}
	}

	return errs
}

func isEmpty(value interface{}) bool {

	if value == nil {
		return true
	}

	switch v := value.(type) {
	case Nullable:
		return v.IsNull()
	case string:
		return v == ""
	case int:
	case int8:
	case int16:
	case int32:
	case int64:
	case uint:
	case uint8:
	case uint16:
	case uint32:
	case uint64:
	case float32:
	case float64:
		return v == 0
	}

	return false
}
