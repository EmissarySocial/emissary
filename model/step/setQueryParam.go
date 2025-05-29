package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// SetQueryParam is a Step that forwards the user to a new page.
type SetQueryParam struct {
	Values map[string]*template.Template
}

// NewSetQueryParam returns a fully initialized SetQueryParam object
func NewSetQueryParam(stepInfo mapof.Any) (SetQueryParam, error) {

	result := SetQueryParam{
		Values: make(map[string]*template.Template),
	}

	for key, value := range stepInfo {
		if key != "do" {
			valueTemplate, err := template.New("value").Parse(convert.String(value))

			if err != nil {
				return SetQueryParam{}, derp.Wrap(err, "model.step.NewSetQueryParam", "Error parsing template", key, value)
			}
			result.Values[key] = valueTemplate
		}
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetQueryParam) Name() string {
	return "set-query-param"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetQueryParam) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetQueryParam) RequiredRoles() []string {
	return []string{}
}
