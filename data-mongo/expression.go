package mongodb

import (
	"github.com/benpate/exp"
	"go.mongodb.org/mongo-driver/bson"
)

// ExpressionToBSON converts a data.Expression value into pure bson.
func ExpressionToBSON(criteria exp.Expression) bson.M {

	switch c := criteria.(type) {

	case exp.Predicate:

		result := bson.M{}
		result[c.Field] = bson.M{operatorBSON(c.Operator): c.Value}
		return result

	case exp.AndExpression:

		if len(c) == 0 {
			return nil
		}

		array := bson.A{}

		for _, exp := range c {
			array = append(array, ExpressionToBSON(exp))
		}

		return bson.M{"$and": array}

	case exp.OrExpression:

		if len(c) == 0 {
			return nil
		}

		array := bson.A{}

		for _, exp := range c {
			array = append(array, ExpressionToBSON(exp))
		}

		return bson.M{"$or": array}
	}

	return bson.M{}
}

// operatorBSON converts a standard data.Operator into the operators used by mongodb
func operatorBSON(operator string) string {

	switch operator {
	case exp.OperatorEqual:
		return "$eq"
	case exp.OperatorNotEqual:
		return "$ne"
	case exp.OperatorLessThan:
		return "$lt"
	case exp.OperatorLessOrEqual:
		return "$le"
	case exp.OperatorGreaterOrEqual:
		return "$ge"
	case exp.OperatorGreaterThan:
		return "$gt"
	default:
		return "$eq"
	}
}
