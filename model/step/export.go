package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Export is an action that can add new model objects of any type
type Export struct {
	Depth       int
	Attachments bool
}

// NewExport returns a fully initialized Export record
func NewExport(stepInfo mapof.Any) (Export, error) {

	// Success
	return Export{
		Depth:       stepInfo.GetInt("depth"),
		Attachments: stepInfo.GetBool("attachments"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Export) AmStep() {}
