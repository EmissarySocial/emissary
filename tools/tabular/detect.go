package tabular

import (
	"bytes"
	"mime"
	"net/url"
	"path"
	"strings"
)

// Detect returns the format of a data file.  It trusts a specific Content-Type first, then the
// extension of the file name or URL, and then the content itself.  Each hint may be empty.
func Detect(contentType string, name string, body []byte) Format {

	// Use the Content-Type, unless it is vague (like text/plain)
	if format := formatFromContentType(contentType); format != FormatUnknown {
		return format
	}

	// Use the file extension, which can be part of a URL
	if format := formatFromName(name); format != FormatUnknown {
		return format
	}

	// Fall back to looking at the content
	return formatFromContent(body)
}

// formatFromContentType returns the format that a Content-Type names, or FormatUnknown
func formatFromContentType(contentType string) Format {

	mediaType, _, err := mime.ParseMediaType(contentType)

	if err != nil {
		return FormatUnknown
	}

	switch mediaType {

	case "text/csv", "application/csv":
		return FormatCSV

	case "text/tab-separated-values":
		return FormatTSV

	case "application/json", "text/json":
		return FormatJSON

	case "application/yaml", "application/x-yaml", "text/yaml", "text/x-yaml":
		return FormatYAML
	}

	return FormatUnknown
}

// formatFromName returns the format that a file name or URL's extension names, or FormatUnknown
func formatFromName(name string) Format {

	// A URL's extension is on its path, before any query string
	if parsed, err := url.Parse(name); err == nil {
		name = parsed.Path
	}

	switch strings.ToLower(path.Ext(name)) {

	case ".csv":
		return FormatCSV

	case ".tsv", ".tab":
		return FormatTSV

	case ".json":
		return FormatJSON

	case ".yaml", ".yml":
		return FormatYAML
	}

	return FormatUnknown
}

// formatFromContent guesses the format from the first meaningful line of the content
func formatFromContent(body []byte) Format {

	firstLine := firstMeaningfulLine(body)

	if len(firstLine) == 0 {
		return FormatUnknown
	}

	// JSON is checked before YAML, because YAML also accepts JSON
	if (firstLine[0] == '[') || (firstLine[0] == '{') {
		return FormatJSON
	}

	// A YAML sequence starts with a dash, or with a document marker
	if (firstLine[0] == '-') || bytes.HasPrefix(firstLine, []byte("---")) {
		return FormatYAML
	}

	// Otherwise, it is a header row.  More tabs than commas means TSV.
	if bytes.Count(firstLine, []byte("\t")) > bytes.Count(firstLine, []byte(",")) {
		return FormatTSV
	}

	return FormatCSV
}

// firstMeaningfulLine returns the first line that is not blank and is not a YAML comment
func firstMeaningfulLine(body []byte) []byte {

	body = bytes.TrimPrefix(body, []byte(byteOrderMark))

	for len(body) > 0 {

		line, rest, _ := bytes.Cut(body, []byte("\n"))
		body = rest
		line = bytes.TrimSpace(line)

		if (len(line) == 0) || (line[0] == '#') {
			continue
		}

		return line
	}

	return nil
}
