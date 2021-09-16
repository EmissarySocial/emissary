package exp

// All a syntactic sugar alias for And(), so that expressions that query all values in a dataset read nicely.
func All() AndExpression {

	return And()
}
