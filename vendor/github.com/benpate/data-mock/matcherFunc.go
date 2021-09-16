package mockdb

import (
	"reflect"

	"github.com/benpate/compare"
	"github.com/benpate/data"
	"github.com/benpate/exp"
)

// MatcherFunc is a helper function that uses reflection to look inside a generic data.Object and match it.
// Because it uses reflection, it should be considered SLOW, and only be used in the mock library.
func MatcherFunc(object data.Object) exp.MatcherFunc {

	return func(predicate exp.Predicate) bool {

		value := reflect.Indirect(reflect.ValueOf(object))
		structure := value.Type()

		_, field, ok := findField(structure, value, predicate.Field)

		if !ok {
			return false
		}

		result, _ := compare.WithOperator(field.Interface(), predicate.Operator, predicate.Value)

		return result
	}
}
