package step

import (
	"github.com/benpate/datatype"
)

// SetSimpleSharing represents an action that can edit a top-level folder in the Domain
type SetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

// NewSetSimpleSharing returns a fully parsed SetSimpleSharing object
func NewSetSimpleSharing(stepInfo datatype.Map) (SetSimpleSharing, error) {

	return SetSimpleSharing{
		Title:   stepInfo.GetString("title"),
		Message: stepInfo.GetString("message"),
		Roles:   stepInfo.GetSliceOfString("roles"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetSimpleSharing) AmStep() {}
