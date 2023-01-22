package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/maps"
)

// SetHeader represents an action-step that can update the custom data stored in a Stream
type SetHeader struct {
	On    string
	Name  string
	Value *template.Template
}

// NewSetHeader returns a fully initialized SetHeader object
func NewSetHeader(stepInfo maps.Map) (SetHeader, error) {

	value, err := template.New("").Parse(getValue(stepInfo.GetString("value")))

	if err != nil {
		return SetHeader{}, derp.Wrap(err, "step.NewSetHeader", "Error parsing value template", value)
	}

	return SetHeader{
		On:    first.String(getValue(stepInfo.GetString("on")), "both"),
		Name:  getValue(stepInfo.GetString("name")),
		Value: value,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetHeader) AmStep() {}
