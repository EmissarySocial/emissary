package tabular

import (
	"bytes"
	"sort"
)

// lineIndex finds the line number of a byte offset in a file
type lineIndex []int

// newLineIndex returns the offsets of every newline in the body
func newLineIndex(body []byte) lineIndex {

	result := make(lineIndex, 0, bytes.Count(body, []byte("\n")))

	for offset, character := range body {
		if character == '\n' {
			result = append(result, offset)
		}
	}

	return result
}

// lineAt returns the line, starting at 1, that holds the provided byte offset
func (index lineIndex) lineAt(offset int64) int {
	return 1 + sort.Search(len(index), func(i int) bool { return int64(index[i]) >= offset })
}
