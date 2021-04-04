package service

import (
	"github.com/benpate/compare"
	"github.com/benpate/exp"
	"github.com/benpate/path"
)

// matcherFunc returns an expression.MatcherFunc that matches any values
// in a model object that are visible via the path.Getter interface.
func matcherFunc(getter path.Getter) exp.MatcherFunc {

	return func(predicate exp.Predicate) bool {

		p := path.New(predicate.Field)
		field, err := getter.GetPath(p)

		if err != nil {
			return false
		}

		result, err := compare.WithOperator(field, predicate.Operator, predicate.Value)

		if err != nil {
			return false
		}

		return result
	}
}
