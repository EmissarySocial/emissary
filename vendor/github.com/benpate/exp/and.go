package exp

// AndExpression combines a series of sub-expressions using AND logic
type AndExpression []Expression

// And combines one or more expression parameters into an AndExpression
func And(expressions ...Expression) AndExpression {

	result := AndExpression{}

	// Add each expression into our result one at a time.
	for _, item := range expressions {
		result = result.Add(item)
	}

	return result
}

// Add appends a new expression into this compound expression
func (andExpression AndExpression) Add(exp Expression) AndExpression {

	// If we're adding another AndExpression to this one, then we can simply concatenate its individual values
	if exp, ok := exp.(AndExpression); ok {
		return append(andExpression, exp...)
	}

	// Fall through to here means that we need to wrap the sub-expression as a single value.
	return append(andExpression, exp)
}

// And allows an additional predicate into this AndExpression
func (andExpression AndExpression) And(name string, operator string, value interface{}) AndExpression {
	return andExpression.Add(New(name, operator, value))
}

func (andExpression AndExpression) AndEqual(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorEqual, value)
}

func (andExpression AndExpression) AndNotEqual(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorNotEqual, value)
}

func (andExpression AndExpression) AndLessThan(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorLessThan, value)
}

func (andExpression AndExpression) AndLessOrEqual(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorLessOrEqual, value)
}

func (andExpression AndExpression) AndGreaterThan(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorGreaterThan, value)
}

func (andExpression AndExpression) AndGreaterOrEqual(name string, value interface{}) AndExpression {
	return andExpression.And(name, OperatorGreaterOrEqual, value)
}

// Match implements the Expression interface.  It loops through all sub-expressions and returns TRUE if all of them match
func (andExpression AndExpression) Match(fn MatcherFunc) bool {

	for _, expression := range andExpression {

		if expression.Match(fn) == false {
			return false
		}
	}

	return true
}
