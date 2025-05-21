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

// AmStep is here only to verify that this struct is a build pipeline step
func (step IfCondition) AmStep() {}
