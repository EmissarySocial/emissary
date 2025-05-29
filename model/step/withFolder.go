package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// WithFolder is a Step that returns a new Folder Builder
type WithFolder struct {
	SubSteps []Step
}

// NewWithFolder returns a fully initialized WithFolder object
func NewWithFolder(stepInfo mapof.Any) (WithFolder, error) {

	const location = "model.step.NewWithFolder"

	subSteps, err := NewPipeline(convert.SliceOfMap(stepInfo["steps"]))

	if err != nil {
		return WithFolder{}, derp.Wrap(err, location, "Invalid 'steps'", stepInfo)
	}

	return WithFolder{
		SubSteps: subSteps,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step WithFolder) Name() string {
	return "with-folder"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step WithFolder) RequiredStates() []string {
	return []string{} // removing this because states may be different in the child objects // requiredStates(step.SubSteps...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step WithFolder) RequiredRoles() []string {
	return requiredRoles(step.SubSteps...)
}
