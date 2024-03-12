package step

import "github.com/benpate/rosetta/mapof"

// RefreshPage represents an pipeline-step that forwards the user to a new page.
type RefreshPage struct{}

// NewRefreshPage returns a fully initialized RefreshPage object
func NewRefreshPage(stepInfo mapof.Any) (RefreshPage, error) {
	return RefreshPage{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step RefreshPage) AmStep() {}
