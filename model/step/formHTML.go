package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
)

// Form represents an action-step that can update the data.DataMap custom data stored in a Stream
type Form struct {
	Form    form.Element
	Options []string
}

// NewForm returns a fully initialized Form object
func NewForm(stepInfo maps.Map) (Form, error) {

	f, err := form.Parse(stepInfo.GetInterface("form"))

	if err != nil {
		return Form{}, derp.Wrap(err, "model.step.NewForm", "Invalid 'form'", stepInfo)
	}

	return Form{
		Form:    f,
		Options: stepInfo.GetSliceOfString("options"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Form) AmStep() {}
