package mongodb

import (
	"encoding/json"
	"testing"

	"github.com/benpate/exp"
	"github.com/stretchr/testify/assert"
)

func TestExpression(t *testing.T) {

	// toJSON converts values into an easy-to-test JSON string
	toJSON := func(value interface{}) string {

		result, err := json.Marshal(value)

		if err != nil {
			return err.Error()
		}

		return string(result)
	}

	{
		// Test combining operators into a single bson.M
		pred := exp.GreaterThan("age", 42)
		assert.Equal(t, toJSON(ExpressionToBSON(pred)), `{"age":{"$gt":42}}`)

		pred2 := pred.And("createDate", exp.OperatorEqual, 10)
		assert.Equal(t, toJSON(ExpressionToBSON(pred2)), `{"$and":[{"age":{"$gt":42}},{"createDate":{"$eq":10}}]}`)

		pred3 := pred2.And("createDate", exp.OperatorLessThan, 20)
		assert.Equal(t, toJSON(ExpressionToBSON(pred3)), `{"$and":[{"age":{"$gt":42}},{"createDate":{"$eq":10}},{"createDate":{"$lt":20}}]}`)
	}

	{
		pred4 := exp.Or(
			exp.New("name", "=", "John Connor").And("favorite_color", "=", "blue"),
			exp.New("name", "=", "Sara Connor").And("favorite_color", "=", "green"),
		)

		assert.Equal(t, toJSON(ExpressionToBSON(pred4)), `{"$or":[{"$and":[{"name":{"$eq":"John Connor"}},{"favorite_color":{"$eq":"blue"}}]},{"$and":[{"name":{"$eq":"Sara Connor"}},{"favorite_color":{"$eq":"green"}}]}]}`)
	}

	{
		pred5 := exp.New("name", "=", "John Connor").Or("favorite_color", "=", "blue")
		assert.Equal(t, toJSON(ExpressionToBSON(pred5)), `{"$or":[{"name":{"$eq":"John Connor"}},{"favorite_color":{"$eq":"blue"}}]}`)
	}

	{
		pred6 := exp.And(
			exp.New("name", "=", "John Connor").Or("favorite_color", "=", "blue"),
			exp.New("name", "=", "Sara Connor").Or("favorite_color", "=", "green"),
		)

		assert.Equal(t, toJSON(ExpressionToBSON(pred6)), `{"$and":[{"$or":[{"name":{"$eq":"John Connor"}},{"favorite_color":{"$eq":"blue"}}]},{"$or":[{"name":{"$eq":"Sara Connor"}},{"favorite_color":{"$eq":"green"}}]}]}`)

	}
	/*
		// Test that all operators are translated correctly.
		ops := exp.New{
		ops.Add("=", exp.OperatorEqual, 0)
		ops.Add("!=", exp.OperatorNotEqual, 0)
		ops.Add("<", exp.OperatorLessThan, 0)
		ops.Add("<=", exp.OperatorLessOrEqual, 0)
		ops.Add(">", exp.OperatorGreaterThan, 0)
		ops.Add(">=", exp.OperatorGreaterOrEqual, 0)
		ops.Add("OTHER", "OTHER", 0)

		assert.Equal(t, "=", ops[0].Name)
		assert.Equal(t, "=", ops[0].Operator)

		assert.Equal(t, "!=", ops[1].Name)
		assert.Equal(t, "!=", ops[1].Operator)

		assert.Equal(t, "<", ops[2].Name)
		assert.Equal(t, "<", ops[2].Operator)

		assert.Equal(t, "<=", ops[3].Name)
		assert.Equal(t, "<=", ops[3].Operator)

		assert.Equal(t, ">", ops[4].Name)
		assert.Equal(t, ">", ops[4].Operator)

		assert.Equal(t, ">=", ops[5].Name)
		assert.Equal(t, ">=", ops[5].Operator)

		assert.Equal(t, "OTHER", ops[6].Name)
		assert.Equal(t, "=", ops[6].Operator)
	*/
}
