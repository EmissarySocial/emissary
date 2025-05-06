package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Include is a Step that calls anoter action to continue processing
type Include struct {
	Action string
}

// NewInclude returns a fully initialized Include object
func NewInclude(stepInfo mapof.Any) (Include, error) {
	return Include{
		Action: stepInfo.GetString("action"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step Include) AmStep() {}
