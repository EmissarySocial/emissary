package tabular

import (
	"bytes"
	"errors"
	"io"
	"strconv"

	"github.com/benpate/derp"
	"go.yaml.in/yaml/v3"
)

// parseYAML reads a single YAML document holding a sequence of mappings
func parseYAML(body []byte, limits Limits) (Table, error) {

	const location = "tabular.parseYAML"

	collector := newCollector(FormatYAML, limits)
	decoder := yaml.NewDecoder(bytes.NewReader(body))

	// An empty file has no rows, which is not an error here
	var document yaml.Node

	if err := decoder.Decode(&document); err != nil {

		if errors.Is(err, io.EOF) {
			return collector.table, nil
		}

		return Table{}, derp.BadRequest(location, "File is not valid YAML", err.Error())
	}

	// RULE: The file must hold one document.  A second one would be ignored without a word.
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Table{}, derp.BadRequest(location, "File must hold a single YAML document")
	}

	// A document always holds one node, but guard the index rather than trust the library
	if len(document.Content) == 0 {
		return collector.table, nil
	}

	// An empty document, such as a lone "---", has no rows
	root := document.Content[0]

	if (root.Kind == yaml.ScalarNode) && (root.Tag == "!!null") {
		return collector.table, nil
	}

	// RULE: The document must be a sequence.  A wrapper mapping such as {items: [...]} is refused.

	if root.Kind != yaml.SequenceNode {
		return Table{}, derp.BadRequest(location, "File must be a YAML list of objects")
	}

	// Read each entry
	for index, node := range root.Content {

		rowNumber := index + 1

		// RULE: The file must fit within the row limit
		if err := collector.checkRowCount(rowNumber); err != nil {
			return Table{}, err
		}

		row, problem, ok := makeYAMLRow(node, rowNumber, limits)

		if !ok {

			if !collector.addProblem(problem) {
				return collector.table, nil
			}

			continue
		}

		collector.addRow(row)
	}

	return collector.table, nil
}

// makeYAMLRow turns one YAML entry into a Row, or describes why it cannot be one
func makeYAMLRow(node *yaml.Node, rowNumber int, limits Limits) (Row, Problem, bool) {

	value, message := yamlValue(node, 1, limits.MaxDepth)

	if message != "" {
		return Row{}, Problem{Row: rowNumber, Line: node.Line, Message: message}, false
	}

	return makeRow(value, rowNumber, node.Line, limits)
}

// yamlValue converts a YAML node into the same values that the JSON reader produces.  Mappings
// become orderedObjects, so that their keys keep the file's order.
func yamlValue(node *yaml.Node, depth int, maxDepth int) (any, string) {

	// RULE: Nodes cannot nest without bound.  This only stops the recursion; it allows one level
	// more than the limit, so that the flattener can report the problem with its column name.
	if depth > maxDepth+2 {
		return nil, "Value is nested more than " + strconv.Itoa(maxDepth) + " levels deep"
	}

	switch node.Kind {

	// RULE: Aliases are refused, because expanding them can multiply a small file without limit
	case yaml.AliasNode:
		return nil, "YAML aliases are not supported"

	case yaml.ScalarNode:
		var result any

		if err := node.Decode(&result); err != nil {
			return nil, "Value cannot be read: " + err.Error()
		}

		return result, ""

	case yaml.MappingNode:
		object := make(orderedObject, 0, len(node.Content)/2)

		for index := 0; index+1 < len(node.Content); index += 2 {

			keyNode := node.Content[index]

			// RULE: Keys must be plain values, not lists, objects, or aliases
			if keyNode.Kind != yaml.ScalarNode {
				return nil, "Keys must be plain text"
			}

			value, message := yamlValue(node.Content[index+1], depth+1, maxDepth)

			if message != "" {
				return nil, message
			}

			object = append(object, field{key: keyNode.Value, value: value})
		}

		return object, ""

	case yaml.SequenceNode:
		array := make([]any, 0, len(node.Content))

		for _, item := range node.Content {

			value, message := yamlValue(item, depth+1, maxDepth)

			if message != "" {
				return nil, message
			}

			array = append(array, value)
		}

		return array, ""
	}

	// Document nodes appear only at the top, which parseYAML has already unwrapped
	return nil, "Value cannot be read"
}
