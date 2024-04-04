package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetHeader represents an action-step that can update the custom data stored in a Stream
type SetHeader struct {
	Method string
	Name   string
	Value  *template.Template
}

// NewSetHeader returns a fully initialized SetHeader object
func NewSetHeader(stepInfo mapof.Any) (SetHeader, error) {

	value, err := template.New("").Parse(stepInfo.GetString("value"))

	if err != nil {
		return SetHeader{}, derp.Wrap(err, "step.NewSetHeader", "Error parsing value template", value)
	}

	return SetHeader{
		Method: first(stepInfo.GetString("method"), "both"),
		Name:   stepInfo.GetString("name"),
		Value:  value,
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetHeader) AmStep() {}
