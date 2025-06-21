package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// AddModelObject is an action that can add new model objects of any type
type AddModelObject struct {
	Form     form.Element
	Defaults []Step
}

// NewAddModelObject returns a fully initialized AddModelObject record
func NewAddModelObject(stepInfo mapof.Any) (AddModelObject, error) {

	// Parse form
	f, err := form.Parse(stepInfo.GetAny("form"))

	if err != nil {
		return AddModelObject{}, derp.Wrap(err, "model.step.NewAddModelObject", "Invalid form", stepInfo["form"])
	}

	// Parse default pipeline
	defaults, err := NewPipeline(stepInfo.GetSliceOfMap("defaults"))

	if err != nil {
		return AddModelObject{}, derp.Wrap(err, "model.step.NewAddModelObject", "Invalid defaults", stepInfo["defaults"])
	}

	// Success
	return AddModelObject{
		Form:     f,
		Defaults: defaults,
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step AddModelObject) Name() string {
	return "add"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step AddModelObject) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step AddModelObject) RequiredStates() []string {
	return requiredStates(step.Defaults...)
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step AddModelObject) RequiredRoles() []string {
	return []string{}
}
