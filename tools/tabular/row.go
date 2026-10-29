package tabular

// Row is one record from a data file, with its cells in the order the file gave them
type Row struct {
	Number int    // Position among the file's rows, starting at 1
	Line   int    // Line in the file where the row begins, starting at 1
	Cells  []Cell // Values in the file's order, each with a unique Path
}

// Get returns the value at the provided path, and whether the row has one
func (row Row) Get(path string) (any, bool) {

	for _, cell := range row.Cells {
		if cell.Path == path {
			return cell.Value, true
		}
	}

	return nil, false
}
