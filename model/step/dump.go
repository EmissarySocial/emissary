package step

import (
	"text/template"

	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
)

// Dump is a Step that can update the data.DataMap custom data stored in a Stream
type Dump struct {
	Value *template.Template
}

// NewDump returns a fully initialized Dump object
func NewDump(stepInfo mapof.Any) (Dump, error) {

	const location = "model.step.NewDump"

	// Parse "value" property
	value, err := template.New("").Parse(stepInfo.GetString("value"))

	if err != nil {
		return Dump{}, derp.Wrap(err, location, "Invalid 'condition'", stepInfo)
	}

	result := Dump{
		Value: value,
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Dump) AmStep() {}
