package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Dump is a Step that can update the data.DataMap custom data stored in a Stream
type Dump struct {
	Value *template.Template
}

// NewDump returns a fully initialized Dump object
func NewDump(stepInfo mapof.Any) (Dump, error) {

	const location = "model.step.NewDump"

	// Parse "value" property
	value, err := template.New("").Parse(stepInfo.GetString("value"))

	if err != nil {
		return Dump{}, derp.Wrap(err, location, "Invalid 'condition'", stepInfo)
	}

	result := Dump{
		Value: value,
	}

	return result, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Dump) Name() string {
	return "dump"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Dump) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Dump) RequiredRoles() []string {
	return []string{}
}
