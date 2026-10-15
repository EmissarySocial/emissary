package tabular

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strconv"

	"github.com/benpate/derp"
)

// parseJSON reads a JSON array of objects, keeping each object's keys in the file's order
func parseJSON(body []byte, limits Limits) (Table, error) {

	const location = "tabular.parseJSON"

	collector := newCollector(FormatJSON, limits)
	lines := newLineIndex(body)
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	// An empty file has no rows, which is not an error here
	token, err := decoder.Token()

	if errors.Is(err, io.EOF) {
		return collector.table, nil
	}

	if err != nil {
		return Table{}, derp.BadRequest(location, "File is not valid JSON", lineMessage(lines, decoder.InputOffset(), err))
	}

	// RULE: The file must be an array.  A wrapper object such as {"items": [...]} is refused.
	if delimiter, isDelimiter := token.(json.Delim); !isDelimiter || (delimiter != '[') {
		return Table{}, derp.BadRequest(location, "File must be a JSON array of objects")
	}

	// Read each entry
	for rowNumber := 1; decoder.More(); rowNumber++ {

		// RULE: The file must fit within the row limit
		if err := collector.checkRowCount(rowNumber); err != nil {
			return Table{}, err
		}

		value, line, err := decodeJSONValue(decoder, lines)

		if err != nil {
			return Table{}, derp.BadRequest(location, "File is not valid JSON", lineMessage(lines, decoder.InputOffset(), err))
		}

		row, problem, ok := makeRow(value, rowNumber, line, limits)

		if !ok {

			if !collector.addProblem(problem) {
				return collector.table, nil
			}

			continue
		}

		collector.addRow(row)
	}

	// Consume the closing bracket, and require nothing after it
	if _, err := decoder.Token(); err != nil {
		return Table{}, derp.BadRequest(location, "File is not valid JSON", lineMessage(lines, decoder.InputOffset(), err))
	}

	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Table{}, derp.BadRequest(location, "File has content after the JSON array")
	}

	return collector.table, nil
}

// lineMessage describes a decoding error, with the line it happened on
func lineMessage(lines lineIndex, offset int64, err error) string {
	return "line " + strconv.Itoa(lines.lineAt(offset)) + ": " + derp.RootCause(err).Error()
}

// decodeJSONValue reads one value from the decoder, and returns it with the line it began on.
// Objects become orderedObjects, so that their keys keep the file's order.
func decodeJSONValue(decoder *json.Decoder, lines lineIndex) (any, int, error) {

	const location = "tabular.decodeJSONValue"

	token, err := decoder.Token()

	if err != nil {
		return nil, 0, derp.Wrap(err, location, "Reading token")
	}

	// The offset now sits just past the value's first token
	line := lines.lineAt(decoder.InputOffset() - 1)

	delimiter, isDelimiter := token.(json.Delim)

	if !isDelimiter {
		return jsonScalar(token), line, nil
	}

	switch delimiter {

	case '{':
		object := make(orderedObject, 0)

		for decoder.More() {

			keyToken, err := decoder.Token()

			if err != nil {
				return nil, 0, derp.Wrap(err, location, "Reading key")
			}

			// Object keys are always strings, as the decoder guarantees
			key, _ := keyToken.(string)
			value, _, err := decodeJSONValue(decoder, lines)

			if err != nil {
				return nil, 0, err
			}

			object = append(object, field{key: key, value: value})
		}

		if _, err := decoder.Token(); err != nil {
			return nil, 0, derp.Wrap(err, location, "Reading end of object")
		}

		return object, line, nil

	case '[':
		array := make([]any, 0)

		for decoder.More() {

			value, _, err := decodeJSONValue(decoder, lines)

			if err != nil {
				return nil, 0, err
			}

			array = append(array, value)
		}

		if _, err := decoder.Token(); err != nil {
			return nil, 0, derp.Wrap(err, location, "Reading end of array")
		}

		return array, line, nil
	}

	// Closing delimiters cannot start a value, because the decoder rejects them first
	return nil, 0, derp.BadRequest(location, "Unexpected delimiter", delimiter.String())
}

// jsonScalar converts a JSON number into an int64 or float64, and returns other scalars as they are
func jsonScalar(token json.Token) any {

	number, isNumber := token.(json.Number)

	if !isNumber {
		return token
	}

	if integer, err := number.Int64(); err == nil {
		return integer
	}

	if float, err := number.Float64(); err == nil {
		return float
	}

	// A number too large for a float64 keeps its digits
	return number.String()
}
