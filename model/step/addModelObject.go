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

// AmStep is here only to verify that this struct is a build pipeline step
func (step AddModelObject) AmStep() {}
