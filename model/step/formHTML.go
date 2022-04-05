package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/form"
)

// Form represents an action-step that can update the data.DataMap custom data stored in a Stream
type Form struct {
	Form form.Form
}

// NewForm returns a fully initialized Form object
func NewForm(stepInfo datatype.Map) (Form, error) {

	f, err := form.Parse(stepInfo.GetInterface("form"))

	if err != nil {
		return Form{}, derp.Wrap(err, "model.step.NewForm", "Invalid 'form'", stepInfo)
	}

	return Form{
		Form: f,
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step Form) AmStep() {}
