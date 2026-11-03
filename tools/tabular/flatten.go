package tabular

import "strconv"

// orderedObject is a JSON or YAML object whose keys keep the order they had in the file
type orderedObject []field

// field is one key and its value in an orderedObject
type field struct {
	key   string
	value any
}

// makeRow turns one JSON or YAML entry into a Row, or describes why it cannot be one
func makeRow(value any, rowNumber int, line int, limits Limits) (Row, Problem, bool) {

	problem := Problem{Row: rowNumber, Line: line}

	// RULE: Every entry must be an object
	object, isObject := value.(orderedObject)

	if !isObject {
		problem.Message = "Entry must be an object"
		return Row{}, problem, false
	}

	cells, column, message := flattenRow(object, limits)

	if message != "" {
		problem.Column = column
		problem.Message = message
		return Row{}, problem, false
	}

	return Row{Number: rowNumber, Line: line, Cells: cells}, Problem{}, true
}

// flattener turns one nested object into the cells of a single row
type flattener struct {
	limits Limits
	cells  []Cell
	seen   map[string]struct{}
}

// flattenRow turns an object into cells, joining nested keys into dotted paths.  It returns the
// column and message of the first problem it finds, if any.
func flattenRow(object orderedObject, limits Limits) ([]Cell, string, string) {

	f := flattener{
		limits: limits,
		cells:  make([]Cell, 0, len(object)),
		seen:   make(map[string]struct{}, len(object)),
	}

	if column, message := f.addObject("", object, 1); message != "" {
		return nil, column, message
	}

	return f.cells, "", ""
}

// addObject adds every field of an object under the provided path prefix
func (f *flattener) addObject(prefix string, object orderedObject, depth int) (string, string) {

	for _, field := range object {

		path := joinPath(prefix, field.key)

		// RULE: Keys cannot be empty, or the path would have an empty segment
		if field.key == "" {
			return path, "Key is empty"
		}

		// Nested objects become longer paths
		if nested, isObject := field.value.(orderedObject); isObject {

			// RULE: Objects cannot nest past the depth limit
			if depth >= f.limits.MaxDepth {
				return path, "Value is nested more than " + strconv.Itoa(f.limits.MaxDepth) + " levels deep"
			}

			if column, message := f.addObject(path, nested, depth+1); message != "" {
				return column, message
			}

			continue
		}

		// Anything else is one cell
		if message := f.addCell(path, field.value, depth); message != "" {
			return path, message
		}
	}

	return "", ""
}

// joinPath joins a prefix and a key into a dotted path
func joinPath(prefix string, key string) string {

	if prefix == "" {
		return key
	}

	return prefix + "." + key
}

// addCell checks one value and appends it as a cell
func (f *flattener) addCell(path string, value any, depth int) string {

	// RULE: Each path can appear only once, even when spelled two ways
	if _, exists := f.seen[path]; exists {
		return "Path appears more than once"
	}

	// RULE: The row must fit within the column limit
	if len(f.cells) >= f.limits.MaxColumns {
		return "Row has more than " + strconv.Itoa(f.limits.MaxColumns) + " columns"
	}

	// RULE: Values inside the cell must fit within the size and depth limits
	value, message := f.checkValue(value, depth)

	if message != "" {
		return message
	}

	f.seen[path] = struct{}{}
	f.cells = append(f.cells, Cell{Path: path, Value: value})
	return ""
}

// checkValue enforces the cell and depth limits on a value, and turns any object inside an
// array into a map, since only the top-level keys need to keep their order
func (f *flattener) checkValue(value any, depth int) (any, string) {

	switch typed := value.(type) {

	case string:
		if len(typed) > f.limits.MaxCellBytes {
			return nil, "Value is longer than " + strconv.Itoa(f.limits.MaxCellBytes) + " bytes"
		}
		return typed, ""

	case []any:
		if depth >= f.limits.MaxDepth {
			return nil, "Value is nested more than " + strconv.Itoa(f.limits.MaxDepth) + " levels deep"
		}

		result := make([]any, len(typed))

		for index, item := range typed {

			checked, message := f.checkValue(item, depth+1)

			if message != "" {
				return nil, message
			}

			result[index] = checked
		}

		return result, ""

	case orderedObject:
		if depth >= f.limits.MaxDepth {
			return nil, "Value is nested more than " + strconv.Itoa(f.limits.MaxDepth) + " levels deep"
		}

		result := make(map[string]any, len(typed))

		for _, field := range typed {

			checked, message := f.checkValue(field.value, depth+1)

			if message != "" {
				return nil, message
			}

			result[field.key] = checked
		}

		return result, ""
	}

	return value, ""
}
