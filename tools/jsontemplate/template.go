package jsontemplate

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"

	"github.com/benpate/derp"
	"github.com/hjson/hjson-go/v4"
)

type Template struct {
	innerTemplate *template.Template
	strictMode    bool // if TRUE, then use the standard (strict) unmarshaller. (Default is to use https://hjson.github.io )
}

func New(input string, options ...Option) (Template, error) {

	result := Template{}
	innerTemplate, err := template.New("").Parse(beginScript + input + endScript)

	if err != nil {
		return result, derp.Wrap(err, "jsontemplate.New", "Error parsing template", input)
	}

	result.innerTemplate = innerTemplate

	for _, option := range options {
		option(&result)
	}

	return result, nil
}

// Execute uses the parsed template to execute the provided value and populate the result variable.
func (t *Template) Execute(result any, value any) error {

	// Execute the template with the provided value
	var buffer bytes.Buffer
	if err := t.innerTemplate.Execute(&buffer, value); err != nil {
		return derp.Wrap(err, "jsontemplate.Execute", "Error executing template", value)
	}

	// Strip <script> tags from the executed template
	resultText := buffer.String()
	resultText = strings.TrimPrefix(resultText, beginScript)
	resultText = strings.TrimSuffix(resultText, endScript)

	if t.strictMode {

		// Unmarshal JSON into the result variable
		if err := json.Unmarshal([]byte(resultText), &result); err != nil {
			return derp.Wrap(err, "jsontemplate.Execute", "Error unmarshalling JSON", resultText)
		}

	} else {

		if err := hjson.Unmarshal([]byte(resultText), &result); err != nil {
			return derp.Wrap(err, "jsontemplate.Execute", "Error unmarshalling HJSON", resultText)
		}
	}

	return nil
}

// Funcs adds the functions to the template's function map.
// It is a pass-through to the underlying template.Funcs method.
func (t *Template) Funcs(funcMap template.FuncMap) {
	t.innerTemplate.Funcs(funcMap)
}
