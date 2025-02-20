package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// SetRenderData is a Step that can update the custom data stored in a builder
type SetRenderData struct {
	Values map[string]*template.Template // values to set directly into the object
}

// NewSetRenderData returns a fully initialized SetRenderData object
func NewSetRenderData(stepInfo mapof.Any) (SetRenderData, error) {

	result := SetRenderData{
		Values: make(map[string]*template.Template, len(stepInfo)),
	}

	for key, value := range stepInfo {
		if key != "do" {
			valueTemplate, err := template.New("value").Parse(convert.String(value))

			if err != nil {
				return SetRenderData{}, derp.Wrap(err, "model.step.NewSetQueryParam", "Error parsing template", key, value)
			}
			result.Values[key] = valueTemplate
		}
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetRenderData) AmStep() {}
