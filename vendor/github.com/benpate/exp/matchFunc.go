package exp

// MatcherFunc is a function signature that is passed in to the .Match() functions
// of every Expression.  It allows the caller to handle the actual matching independently of their underlying data, while
// the Expression objects handle the program flow.
type MatcherFunc func(Predicate) bool
