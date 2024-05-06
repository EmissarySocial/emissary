package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ViewCSS represents an action-step that can build a Stream into HTML
type ViewCSS struct {
	File string
}

// NewViewCSS generates a fully initialized ViewCSS step.
func NewViewCSS(stepInfo mapof.Any) (ViewCSS, error) {

	return ViewCSS{
		File: stepInfo.GetString("file"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ViewCSS) AmStep() {}
