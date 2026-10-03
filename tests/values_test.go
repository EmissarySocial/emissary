package tests

import (
	"net/url"
	"strings"

	"github.com/benpate/rosetta/schema"
)

// scenario builds the request values for one kind of visitor input
type scenario struct {
	Name   string
	Values func(fields []postField) url.Values
}

// scenarios are applied to every postCase.  Each is deterministic, so that a golden file
// changes only when behavior does.
var scenarios = []scenario{
	{Name: "typical", Values: eachField(typicalValue)},
	{Name: "empty", Values: eachField(func(postField) []string { return []string{""} })},
	{Name: "hostile", Values: eachField(hostileValue)},
	{Name: "absent", Values: func([]postField) url.Values { return url.Values{} }},
}

// eachField returns a scenario that posts one generated value for every field
func eachField(generate func(postField) []string) func([]postField) url.Values {
	return func(fields []postField) url.Values {
		result := url.Values{}
		for _, field := range fields {
			result[field.Path] = generate(field)
		}
		return result
	}
}

// typicalValue returns a well-formed value for a field, the way a visitor would fill it in
func typicalValue(field postField) []string {

	// A value the step offers is the most typical input of all
	if len(field.Enum) > 0 {
		return []string{field.Enum[0]}
	}

	switch field.Widget {
	case "checkbox", "toggle":
		return []string{"true"}
	case "place.formatted":
		return []string{"1 Main Street, Portland, OR"}
	case "place.longitude":
		return []string{"-122.6765"}
	case "place.latitude":
		return []string{"45.5231"}
	}

	switch element := field.Element.(type) {

	case schema.Boolean:
		return []string{"true"}

	case schema.Integer:
		return []string{"7"}

	case schema.Number:
		return []string{"3.5"}

	case schema.Array:
		first := typicalValue(postField{Path: field.Path + ".0", Element: element.Items})
		second := typicalValue(postField{Path: field.Path + ".1", Element: element.Items})
		return append(first, second...)

	case schema.String:
		if len(element.Enum) > 0 {
			return []string{element.Enum[0]}
		}
		return []string{typicalString(field.Path, element.Format)}
	}

	return []string{"Sample " + field.Path}
}

// typicalString returns a well-formed string for a schema format
func typicalString(path string, format string) string {

	switch format {
	case "email":
		return "visitor@example.com"
	case "url", "uri":
		return "https://example.com/" + slug(path)
	case "date":
		return "2026-10-02"
	case "datetime", "date-time":
		return "2026-10-02T15:04:05Z"
	case "time":
		return "15:04"
	case "color":
		return "#336699"
	case "objectId":
		return "0123456789abcdef01234567"
	case "token":
		return "sample-token"
	case "html":
		return "<p>Sample <b>" + path + "</b></p>"
	case "markdown":
		return "# Sample\n\n*" + path + "*"
	case "username":
		return "visitor_name"
	case "regex":
		return "^[a-z]+$"
	case "webfinger":
		return "@visitor@example.com"
	case "css":
		return "body { color: #336699; }"
	case "css-declarations":
		return "color: #336699;"
	}

	return "Sample " + path
}

// hostileValue returns input that a careless or malicious visitor might send
func hostileValue(field postField) []string {

	switch element := field.Element.(type) {

	case schema.Boolean:
		return []string{"maybe"}

	case schema.Integer:
		return []string{"-99999999999999999999"}

	case schema.Number:
		return []string{"NaN"}

	case schema.Array:
		return []string{"<b>x</b>", "", "not-an-objectid"}

	case schema.String:
		switch element.Format {
		case "email":
			return []string{"not an email <script>"}
		case "url", "uri":
			return []string{"javascript:alert(1)"}
		case "date", "datetime", "date-time", "time":
			return []string{"2026-13-45T99:99"}
		case "color":
			return []string{"red; background:url(x)"}
		case "objectId":
			return []string{"not-an-objectid"}
		}
		return []string{hostileString(element.MaxLength)}
	}

	return []string{hostileString(0)}
}

// hostileString returns markup, control characters, and text longer than the field allows
func hostileString(maxLength int) string {

	length := 300
	if maxLength > 0 {
		length = maxLength + 5
	}

	prefix := "<script>alert(1)</script><img src=x onerror=alert(2)>\u202e\t"
	return prefix + strings.Repeat("x", max(length-len([]rune(prefix)), 1))
}

// slug turns a path into something safe to put in a URL
func slug(path string) string {
	return strings.NewReplacer(".", "-", " ", "-").Replace(path)
}
