package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/schema"
)

// StartupSaveTask is a Step that records one completed startup task in the Domain, so that the
// startup wizard can tell which of its Theme's tasks the owner has already worked through.
type StartupSaveTask struct {
	Value string
}

// NewStartupSaveTask returns a fully initialized StartupSaveTask object
func NewStartupSaveTask(stepInfo mapof.Any) (StartupSaveTask, error) {

	const location = "model.step.NewStartupSaveTask"

	// Validate the step configuration
	if _, err := schema.New(StepStartupSaveTaskSchema()).Validate(stepInfo); err != nil {
		return StartupSaveTask{}, derp.Wrap(err, location, "Invalid step configuration", stepInfo)
	}

	return StartupSaveTask{
		Value: stepInfo.GetString("value"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step StartupSaveTask) Name() string {
	return "startup-save-task"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step StartupSaveTask) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step StartupSaveTask) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step StartupSaveTask) RequiredRoles() []string {
	return []string{}
}

// StepStartupSaveTaskSchema returns a validating schema for the StartupSaveTask step.  The
// MaxLength matches the "startupTasks" property in model.DomainSchema(), so an over-long value
// fails when the Template loads instead of when the Domain is saved.
func StepStartupSaveTaskSchema() schema.Element {
	return schema.Object{
		Properties: schema.ElementMap{
			"value": schema.String{Required: true, MinLength: 1, MaxLength: 32},
		},
	}
}
