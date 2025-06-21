package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// IfCondition is a Step that can update the data.DataMap custom data stored in a Stream
type IfCondition struct {
	Condition *template.Template
	Then      []Step
	Otherwise []Step
}

func NewIfCondition(stepInfo mapof.Any) (IfCondition, error) {

	const location = "model.step.NewIfCondition"

	// Parse "condition" property
	condition, err := template.New("").Parse(stepInfo.GetString("condition"))

	if err != nil {
		return IfCondition{}, derp.Wrap(err, location, "Invalid 'condition'", stepInfo)
	}

	// Parse "then" property
	then, err := NewPipeline(stepInfo.GetSliceOfMap("then"))

	if err != nil {
		return IfCondition{}, derp.Wrap(err, location, "Invalid 'then'", stepInfo)
	}

	// Parse "else" property
	otherwise, err := NewPipeline(stepInfo.GetSliceOfMap("else"))

	if err != nil {
		return IfCondition{}, derp.Wrap(err, location, "Invalid 'else'", stepInfo)
	}

	return IfCondition{
		Condition: condition,
		Then:      then,
		Otherwise: otherwise,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step IfCondition) Name() string {
	return "if"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step IfCondition) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step IfCondition) RequiredStates() []string {
	return append(requiredStates(step.Then...), requiredStates(step.Otherwise...)...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step IfCondition) RequiredRoles() []string {
	return append(requiredRoles(step.Then...), requiredRoles(step.Otherwise...)...)
}
