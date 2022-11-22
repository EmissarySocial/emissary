package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/maps"
)

// TableEditor is an action that can add new sub-streams to the domain.
type TableEditor struct {
	Path string
	Form form.Element
}

// NewTableEditor returns a fully initialized TableEditor record
func NewTableEditor(stepInfo maps.Map) (TableEditor, error) {

	f, err := form.Parse(stepInfo.GetInterface("form"))

	if err != nil {
		return TableEditor{}, derp.Wrap(err, "model.step.NewTableEditor", "Invalid 'form'", stepInfo)
	}

	return TableEditor{
		Path: stepInfo.GetString("path"),
		Form: f,
	}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step TableEditor) AmStep() {}
