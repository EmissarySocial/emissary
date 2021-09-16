# Expression Builder

Every database has its own query language, so this library provides in intermediate format that should be easy to convert into whatever specific language you need to use.  

The expression library only represents the structure of the logical expression, and does not include implementations for any data sources.  Those should be implemented in each individual data source adapter library.

```go

// build single predicate expressions
criteria := exp.Equal("_id", 42)

// use chaining for logical constructs
criteria := exp.Equal("_id", 42).AndEqual("deleteDate",  0)
criteria := exp.Equal("name", "John").OrEqual("name", "Sarah")

// Also supports complex and/or logic

criteria := exp.Or(
    exp.Equal("_id", 42).AndEqual("deleteDate",  0),
    exp.Equal("_id", 42).AndEqual("name", "Sarah"),
)

// Constants define standard expected operators
data.OperatorEqual          = "="
data.OperatorNotEqual       = "!="
data.OperatorLessThan       = "<"
data.OperatorLessOrEqual    = "<="
data.OperatorGreaterThan    = ">"
data.OperatorGreaterOrEqual = ">="
```

## Interfaces

This is accomplished with three very similar data types that all implement the same `Expression` interface.

**`Predicate`** represents a single predicate/comparison.  Using `.And()` and `.Or()` will return the corresponding `AndExpression` or `OrExpression` object

**`AndExpression`** represents multiple predicates, all chained together with AND logic.  Only supports the `.And()` method for additional predicates

**`OrExpression`** represents multiple predicates, all chained together with OR logic.  Only supports the `.Or()` method for additional predicates.

## Manually Walking the Logic Tree

Each of the three interfaces above implements a `.Match()` function that can be used by external programs to see if a dataset matches this exp.  You must pass in a `MatcherFunc` that accepts a predicate and returns `TRUE` if that predicate matches the dataaset.  `AndExpression` and `OrExpression` objects will call this function repeatedly for each element in their logic tree, and return a final boolean value for the entire logic structure.

## Pull Requests Welcome

This library is a work in progress, and will benefit from your experience reports, use cases, and contributions.  If you have an idea for making this library better, send in a pull request.  We're all in this together! 📚
