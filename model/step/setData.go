package step

import "github.com/benpate/rosetta/mapof"

// SetData represents an action-step that can update the custom data stored in a Stream
type SetData struct {
	FromURL  []string  // List of paths to pull from URL data
	FromForm []string  // List of paths to pull from Form data
	Values   mapof.Any // values to set directly into the object
	Defaults mapof.Any // values to set into the object IFF they are currently empty.
}

// NewSetData returns a fully initialized SetData object
func NewSetData(stepInfo mapof.Any) (SetData, error) {

	return SetData{
		FromURL:  stepInfo.GetSliceOfString("from-url"),
		FromForm: stepInfo.GetSliceOfString("from-form"),
		Values:   stepInfo.GetMap("values"),
		Defaults: stepInfo.GetMap("defaults"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetData) AmStep() {}
