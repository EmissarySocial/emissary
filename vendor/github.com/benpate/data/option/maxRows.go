package option

// TypeMaxRows is the token that designates the maximum number of records to be returned
const TypeMaxRows = "MAXROWS"

// MaxRowsConfig is a query option that limits the number of rows to be included in a dataset
type MaxRowsConfig int

// MaxRows returns a query option that will limit the query results to a certain number of rows
func MaxRows(maxRows int64) Option {
	return MaxRowsConfig(maxRows)
}

// OptionType identifies this record as a query option
func (maxRowsConfig MaxRowsConfig) OptionType() string {
	return TypeMaxRows
}

// MaxRows returns the maximum number of rows to include in a dataset
func (maxRowsConfig MaxRowsConfig) MaxRows() int {
	return int(maxRowsConfig)
}
