package compare

import (
	"github.com/benpate/derp"
)

// WithOperator uses an operator to compare two values, and returns TRUE or FALSE
func WithOperator(value1 interface{}, operator string, value2 interface{}) (bool, error) {

	// These operations are performed outside of the "Interface" comparison
	switch operator {
	case OperatorBeginsWith:
		return BeginsWith(value1, value2), nil

	case OperatorContains:
		return Contains(value1, value2), nil

	case OperatorEndsWith:
		return EndsWith(value1, value2), nil
	}

	result, err := Interface(value1, value2)

	if err != nil {
		return false, derp.Wrap(err, "compare.WithOperator", "Can't Compare Values", value1, operator, value2)
	}

	switch operator {

	case OperatorGreaterThan:
		return (result == 1), nil

	case OperatorGreaterOrEqual:
		return (result != -1), nil

	case OperatorEqual:
		return (result == 0), nil

	case OperatorLessOrEqual:
		return (result != 1), nil

	case OperatorLessThan:
		return (result == -1), nil

	case OperatorNotEqual:
		return (result != 0), nil

	default:
		return false, derp.New(500, "compare.WithOperator", "Unrecognized Operator", value1, operator, value2)

	}
}

// Equal is a simplified version of Compare.  It ONLY returns true if the two provided values are EQUAL.
// In all other cases (including errors) it returns FALSE
func Equal(value1 interface{}, value2 interface{}) bool {

	if result, err := Interface(value1, value2); err == nil {

		if result == 0 {
			return true
		}
	}

	return false
}

// LessThan is a simplified version of Compare.  It ONLY returns true if value1 is verifiably LESS THAN value2.
// In all other cases (including errors) it returns FALSE
func LessThan(value1 interface{}, value2 interface{}) bool {

	if result, err := Interface(value1, value2); err == nil {

		if result == -1 {
			return true
		}
	}

	return false
}

// GreaterThan is a simplified version of Compare.  It ONLY returns true if value1 is verifiably GREATER THAN value2.
// In all other cases (including errors) it returns FALSE
func GreaterThan(value1 interface{}, value2 interface{}) bool {

	if result, err := Interface(value1, value2); err == nil {

		if result == 1 {
			return true
		}
	}

	return false
}
