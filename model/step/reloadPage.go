package step

import "github.com/benpate/rosetta/maps"

// ReloadPage represents an pipeline-step that forwards the user to a new page.
type ReloadPage struct{}

// NewReloadPage returns a fully initialized ReloadPage object
func NewReloadPage(stepInfo maps.Map) (ReloadPage, error) {
	return ReloadPage{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step ReloadPage) AmStep() {}
