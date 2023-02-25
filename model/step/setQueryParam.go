package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// SetQueryParam represents an action-step that forwards the user to a new page.
type SetQueryParam struct {
	Values map[string]*template.Template
}

// NewSetQueryParam returns a fully initialized SetQueryParam object
func NewSetQueryParam(stepInfo mapof.Any) (SetQueryParam, error) {

	result := SetQueryParam{
		Values: make(map[string]*template.Template),
	}

	for key, value := range stepInfo {
		if key != "step" {
			valueTemplate, err := template.New("value").Parse(convert.String(value))

			if err != nil {
				return SetQueryParam{}, derp.Wrap(err, "model.step.NewSetQueryParam", "Error parsing template", key, value)
			}
			result.Values[key] = valueTemplate
		}
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetQueryParam) AmStep() {}
