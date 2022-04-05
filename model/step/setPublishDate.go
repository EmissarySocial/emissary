package step

import (
	"github.com/benpate/datatype"
)

// SetPublishDate represents an action-step that can update a stream's PublishDate with the current time.
type SetPublishDate struct{}

// NewSetPublishDate returns a fully initialized SetPublishDate object
func NewSetPublishDate(stepInfo datatype.Map) (SetPublishDate, error) {
	return SetPublishDate{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetPublishDate) AmStep() {}
