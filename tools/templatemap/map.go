package templatemap

import (
	"bytes"
	"encoding/json"
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/funcmap"
)

// Map is a collection of named, pre-parsed text templates
type Map map[string]*template.Template

// New returns a fully initialized, empty Map
func New() Map {
	return make(Map)
}

// Execute executes the named template with the specified value, and returns the result as a string.
func (m Map) Execute(name string, value any) string {

	if template, exists := m[name]; exists {
		var buffer bytes.Buffer
		if err := template.Execute(&buffer, value); err == nil {
			return buffer.String()
		} else {
			derp.Report(derp.Wrap(err, "tools.templatemap.Execute", "Executing template", name, value))
		}
	}

	return ""
}

// UnmarshalJSON implements the json.Unmarshaler interface, compiling each JSON string value into a template
func (m *Map) UnmarshalJSON(data []byte) error {

	const location = "tools.templatemap.UnmarshalJSON"

	temp := make(map[string]string)

	if err := json.Unmarshal(data, &temp); err != nil {
		return derp.Wrap(err, location, "Unmarshalling JSON")
	}

	funcMap := funcmap.All()

	for key, value := range temp {
		tmpl, err := template.New(key).Funcs(funcMap).Parse(value)

		if err != nil {
			return derp.Wrap(err, location, "Parsing template", key)
		}

		(*m)[key] = tmpl
	}

	return nil
}
