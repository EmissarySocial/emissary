package step

import (
	"github.com/benpate/derp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/mapof"
)

// TableEditor is an action that can add new sub-streams to the domain.
type TableEditor struct {
	Path string
	Form form.Element
}

// NewTableEditor returns a fully initialized TableEditor record
func NewTableEditor(stepInfo mapof.Any) (TableEditor, error) {

	f, err := form.Parse(stepInfo.GetAny("form"))

	if err != nil {
		return TableEditor{}, derp.Wrap(err, "model.step.NewTableEditor", "Invalid 'form'", stepInfo)
	}

	return TableEditor{
		Path: stepInfo.GetString("path"),
		Form: f,
	}, nil
}

// AmStep is here to verify that this struct is a build pipeline step
func (step TableEditor) AmStep() {}
