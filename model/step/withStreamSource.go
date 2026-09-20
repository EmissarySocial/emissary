package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithStreamSource is a Step that switches to the StreamSource record attached to a Stream,
// creating one if it does not exist yet, and executes its sub-steps against it
type WithStreamSource struct {
	SubSteps []Step
}

// NewWithStreamSource returns a fully initialized WithStreamSource object
func NewWithStreamSource(stepInfo mapof.Any) (WithStreamSource, error) {

	const location = "step.NewWithStreamSource"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithStreamSource{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithStreamSource{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithStreamSource) Name() string {
	return "with-stream-source"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step WithStreamSource) RequiredModel() string {
	return "Stream"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithStreamSource) RequiredStates() []string {
	// RULE: A StreamSource has one state of its own, so a sub-step's state requirement would name
	// a state of the STREAM, which this step has switched away from.
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithStreamSource) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
