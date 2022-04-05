package step

import (
	"github.com/benpate/datatype"
)

// RefreshPage represents an pipeline-step that forwards the user to a new page.
type RefreshPage struct{}

// NewRefreshPage returns a fully initialized RefreshPage object
func NewRefreshPage(stepInfo datatype.Map) (RefreshPage, error) {
	return RefreshPage{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step RefreshPage) AmStep() {}
