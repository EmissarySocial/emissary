package tabular

import (
	"bytes"
	"unicode/utf8"

	"github.com/benpate/derp"
)

// Parse reads a data file in the provided format.  Rows that cannot be read are reported as
// Problems in the Table.  An error means the file as a whole could not be read.
func Parse(body []byte, format Format, limits Limits) (Table, error) {

	const location = "tabular.Parse"

	limits = limits.withDefaults()

	// RULE: The file must fit within the byte limit
	if len(body) > limits.MaxBytes {
		return Table{}, derp.BadRequest(location, "File is larger than the limit", len(body), limits.MaxBytes)
	}

	// RULE: The file must be UTF-8 text.  Other encodings would be stored as garbage.
	if !utf8.Valid(body) {
		return Table{}, derp.BadRequest(location, "File is not UTF-8 text")
	}

	// Spreadsheet exports often begin with a byte-order mark, which would stick to the first heading
	body = bytes.TrimPrefix(body, []byte(byteOrderMark))

	switch format {

	case FormatCSV:
		return parseDelimited(body, ',', format, limits)

	case FormatTSV:
		return parseDelimited(body, '\t', format, limits)

	case FormatJSON:
		return parseJSON(body, limits)

	case FormatYAML:
		return parseYAML(body, limits)
	}

	return Table{}, derp.BadRequest(location, "Unrecognized format", format)
}
