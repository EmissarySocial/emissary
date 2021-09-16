package option

// TypeFirstRow is the token that designates the "row number" to begin results.
const TypeFirstRow = "FIRSTROW"

// FirstRowConfig is a query option that sets the first row to be returned in a dataset
type FirstRowConfig int

// FirstRow returns a query option that will limit the query results to a certain number of rows
func FirstRow(firstRow int64) Option {
	return FirstRowConfig(firstRow)
}

// OptionType identifies this record as a query option
func (firstRowConfig FirstRowConfig) OptionType() string {
	return TypeFirstRow
}

// FirstRow returns the first row of the dataset to return
func (firstRowConfig FirstRowConfig) FirstRow() int {
	return int(firstRowConfig)
}
