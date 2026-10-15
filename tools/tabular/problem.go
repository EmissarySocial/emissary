package tabular

import "strconv"

// Problem describes one row that could not be read, in terms an author can fix in a spreadsheet
type Problem struct {
	Row     int    // Position among the file's rows, starting at 1, or 0 for the header
	Line    int    // Line in the file, starting at 1
	Column  string // Path of the offending column, when there is one
	Message string // What is wrong
}

// String returns the problem as one line, such as `line 4, column "label": Value is too long`
func (problem Problem) String() string {

	result := "line " + strconv.Itoa(problem.Line)

	if problem.Column != "" {
		result += ", column " + strconv.Quote(problem.Column)
	}

	return result + ": " + problem.Message
}
