package exp

// OrExpression compares a series of sub-expressions, using the OR logic
type OrExpression []Expression

// Or combines one or more expression parameters into an OrExpression
func Or(expressions ...Expression) OrExpression {

	result := OrExpression{}

	// Add each expression into our result one at a time.
	for _, item := range expressions {
		result = result.Add(item)
	}

	return result

}

// Add appends a new expression into this compound expression
func (orExpression OrExpression) Add(exp Expression) OrExpression {

	// If we're adding another OrExpression to this one, then we can simply concatenate its individual values
	if exp, ok := exp.(OrExpression); ok {
		return append(orExpression, exp...)
	}

	// Fall through to here means that we need to group its sub-values into a single item
	return append(orExpression, exp)
}

// Or appends an additional predicate into the OrExpression
func (orExpression OrExpression) Or(name string, operator string, value interface{}) OrExpression {
	return orExpression.Add(New(name, operator, value))
}

// Match implements the Expression interface.  It loops through all sub-expressions and returns TRUE if any of them match
func (orExpression OrExpression) Match(fn MatcherFunc) bool {

	for _, expression := range orExpression {

		if expression.Match(fn) == true {
			return true
		}
	}

	return false
}
