package tabular

import (
	"bytes"
	"encoding/csv"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/benpate/derp"
)

// parseDelimited reads CSV or TSV, whose first record is a header of paths
func parseDelimited(body []byte, comma rune, format Format, limits Limits) (Table, error) {

	const location = "tabular.parseDelimited"

	collector := newCollector(format, limits)
	reader := csv.NewReader(bytes.NewReader(body))
	reader.Comma = comma

	// Read the header.  An empty file has no rows, which is not an error here.
	record, err := reader.Read()

	if err != nil {

		if errors.Is(err, io.EOF) {
			return collector.table, nil
		}

		return Table{}, derp.BadRequest(location, "Header cannot be read", err.Error())
	}

	// RULE: The header must fit within the column limit
	if len(record) > limits.MaxColumns {
		return Table{}, derp.BadRequest(location, "File has more than "+strconv.Itoa(limits.MaxColumns)+" columns")
	}

	// RULE: Every heading must be a unique, non-empty path.  Rows mean nothing without them.
	headerLine, _ := reader.FieldPos(0)
	headings, ok := readHeadings(&collector, record, headerLine)

	if !ok {
		return collector.table, nil
	}

	// Read each row.  The reader requires every record to have as many fields as the header.
	for rowNumber := 1; ; rowNumber++ {

		record, err := reader.Read()

		if errors.Is(err, io.EOF) {
			return collector.table, nil
		}

		// RULE: The file must fit within the row limit
		if limitError := collector.checkRowCount(rowNumber); limitError != nil {
			return Table{}, limitError
		}

		if err != nil {

			// A short or long row is a problem in that row alone
			var parseError *csv.ParseError
			if errors.As(err, &parseError) && errors.Is(parseError.Err, csv.ErrFieldCount) {

				if !collector.addProblem(Problem{
					Row:     rowNumber,
					Line:    parseError.StartLine,
					Message: "Row has " + strconv.Itoa(len(record)) + " cells, but the header has " + strconv.Itoa(len(headings)),
				}) {
					return collector.table, nil
				}

				continue
			}

			// Any other syntax error ends the file, because the reader cannot find the next row
			return Table{}, derp.BadRequest(location, "File cannot be read", err.Error())
		}

		// Collect the cells
		line, _ := reader.FieldPos(0)
		row, problem, ok := readDelimitedRow(headings, record, rowNumber, line, limits)

		if !ok {

			if !collector.addProblem(problem) {
				return collector.table, nil
			}

			continue
		}

		collector.addRow(row)
	}
}

// readHeadings validates a header record, and records a Problem for each bad heading
func readHeadings(collector *collector, record []string, line int) ([]string, bool) {

	headings := make([]string, len(record))
	seen := make(map[string]struct{}, len(record))
	ok := true

	for index, heading := range record {

		heading = strings.TrimSpace(heading)
		headings[index] = heading
		problem := Problem{Line: line, Column: heading}

		// RULE: Headings cannot be empty
		if heading == "" {
			problem.Message = "Heading " + strconv.Itoa(index+1) + " is empty"
			ok = false

			if !collector.addProblem(problem) {
				return headings, false
			}

			continue
		}

		// RULE: Headings cannot repeat
		if _, exists := seen[heading]; exists {
			problem.Message = "Heading appears more than once"
			ok = false

			if !collector.addProblem(problem) {
				return headings, false
			}

			continue
		}

		seen[heading] = struct{}{}
	}

	return headings, ok
}

// readDelimitedRow pairs each value in a record with its heading
func readDelimitedRow(headings []string, record []string, rowNumber int, line int, limits Limits) (Row, Problem, bool) {

	row := Row{
		Number: rowNumber,
		Line:   line,
		Cells:  make([]Cell, len(record)),
	}

	for index, value := range record {

		// RULE: Each value must fit within the cell limit
		if len(value) > limits.MaxCellBytes {
			return Row{}, Problem{
				Row:     rowNumber,
				Line:    line,
				Column:  headings[index],
				Message: "Value is longer than " + strconv.Itoa(limits.MaxCellBytes) + " bytes",
			}, false
		}

		row.Cells[index] = Cell{Path: headings[index], Value: value}
	}

	return row, Problem{}, true
}
