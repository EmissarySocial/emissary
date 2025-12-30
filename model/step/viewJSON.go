package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// ViewJSON is a Step that can build a Stream into HTML
type ViewJSON struct {
	Value *template.Template
}

// NewViewJSON generates a fully initialized ViewJSON step.
func NewViewJSON(stepInfo mapof.Any) (ViewJSON, error) {

	const location = "build.NewViewJSON"

	value := stepInfo.GetString("value")

	if value == "" {
		return ViewJSON{}, derp.Validation("Step must require a query template")
	}

	value = "{{" + value + " | json}}"

	if jsonp := stepInfo.GetString("jsonp"); jsonp != "" {
		value = jsonp + "(" + value + ");"
	}

	valueTemplate, err := template.New("").Funcs(FuncMap()).Parse(value)

	if err != nil {
		return ViewJSON{}, derp.Wrap(err, location, "Unable to parse JSON query template")
	}

	return ViewJSON{
		Value: valueTemplate,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step ViewJSON) Name() string {
	return "view-json"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step ViewJSON) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step ViewJSON) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step ViewJSON) RequiredRoles() []string {
	return []string{}
}
