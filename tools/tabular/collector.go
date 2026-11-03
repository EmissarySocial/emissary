package tabular

import (
	"strconv"

	"github.com/benpate/derp"
)

// collector gathers the rows and problems of one file, and enforces the limits that span rows
type collector struct {
	limits Limits
	table  Table
}

// newCollector returns a collector for one file in the provided format
func newCollector(format Format, limits Limits) collector {
	return collector{
		limits: limits,
		table: Table{
			Format:   format,
			Rows:     make([]Row, 0),
			Problems: make([]Problem, 0),
		},
	}
}

// addRow appends a row that was read without a problem
func (c *collector) addRow(row Row) {
	c.table.Rows = append(c.table.Rows, row)
}

// addProblem records a row that could not be read, and returns FALSE once the reader should
// stop because the problem limit is reached
func (c *collector) addProblem(problem Problem) bool {

	c.table.Problems = append(c.table.Problems, problem)

	if len(c.table.Problems) < c.limits.MaxProblems {
		return true
	}

	// Say why the reading stopped, so the author knows more rows may be wrong
	c.table.Problems = append(c.table.Problems, Problem{
		Row:     problem.Row,
		Line:    problem.Line,
		Message: "Too many problems; stopped reading here",
	})

	return false
}

// checkRowCount returns an error when one more row would exceed the row limit
func (c *collector) checkRowCount(rowNumber int) error {

	const location = "tabular.collector.checkRowCount"

	if rowNumber > c.limits.MaxRows {
		return derp.BadRequest(location, "File has more than "+strconv.Itoa(c.limits.MaxRows)+" rows")
	}

	return nil
}
