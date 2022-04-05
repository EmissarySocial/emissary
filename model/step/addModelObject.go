package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// AddModelObject is an action that can add new model objects of any type
type AddModelObject struct {
	Form     form.Form
	Defaults []Step
}

// NewAddModelObject returns a fully initialized AddModelObject record
func NewAddModelObject(stepInfo datatype.Map) (AddModelObject, error) {

	// Parse form
	f, err := form.Parse(stepInfo.GetInterface("form"))

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

// AmStep is here only to verify that this struct is a render pipeline step
func (step AddModelObject) AmStep() {}
