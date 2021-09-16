package option

// TypeSort is the token that designates a Sort order
const TypeSort = "SORT"

// SortDirectionAscending is the token that designates that records should be sorted lowest to highest
const SortDirectionAscending = "ASC"

// SortDirectionDescending is the token that designates that records should be sorted highest to lowest
const SortDirectionDescending = "DESC"

// SortConfig identifies the field and direction to use when sorting a dataset
type SortConfig struct {
	FieldName string
	Direction string
}

// OptionType identifies this record as a query option
func (sortConfig SortConfig) OptionType() string {
	return TypeSort
}

// SortAsc returns a query option that will sort the query results in ASCENDING order
func SortAsc(fieldName string) Option {
	return SortConfig{
		FieldName: fieldName,
		Direction: SortDirectionAscending,
	}
}

// SortDesc returns a query option that will sort the query results in DESCENDING order
func SortDesc(fieldName string) Option {
	return SortConfig{
		FieldName: fieldName,
		Direction: SortDirectionDescending,
	}
}
