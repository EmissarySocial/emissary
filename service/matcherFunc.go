package service

import (
	"github.com/benpate/data/compare"
	"github.com/benpate/data/expression"
	"github.com/benpate/path"
)

// matcherFunc returns an expression.MatcherFunc that matches any values
// in a model object that are visible via the path.Getter interface.
func matcherFunc(getter path.Getter) expression.MatcherFunc {

	return func(predicate expression.Predicate) bool {

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