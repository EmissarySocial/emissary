package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// SetHeader is a Step that can update the custom data stored in a Stream
type SetHeader struct {
	Method     string
	HeaderName string
	Value      *template.Template
}

// NewSetHeader returns a fully initialized SetHeader object
func NewSetHeader(stepInfo mapof.Any) (SetHeader, error) {

	value, err := template.New("").Parse(stepInfo.GetString("value"))

	if err != nil {
		return SetHeader{}, derp.Wrap(err, "step.NewSetHeader", "Error parsing value template", value)
	}

	return SetHeader{
		Method:     first(stepInfo.GetString("method"), "both"),
		HeaderName: stepInfo.GetString("name"),
		Value:      value,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetHeader) Name() string {
	return "set-header"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetHeader) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetHeader) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetHeader) RequiredRoles() []string {
	return []string{}
}
