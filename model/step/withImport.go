package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithImport is a Step that returns a new Import Builder
type WithImport struct {
	SubSteps []Step
}

// NewWithImport returns a fully initialized WithImport object
func NewWithImport(stepInfo mapof.Any) (WithImport, error) {

	const location = "NewWithImport"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithImport{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithImport{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithImport) Name() string {
	return "with-import"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithImport) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithImport) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithImport) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
