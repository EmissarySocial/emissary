package tabular

// Cell is one value in a Row, named by the dotted path it should be written to
type Cell struct {
	Path  string // Dotted path, such as "location.latitude"
	Value any    // A string, int64, float64, bool, nil, []any, or map[string]any
}
