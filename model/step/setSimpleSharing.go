package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

// SetSimpleSharing represents an action that can edit a top-level folder in the Domain
type SetSimpleSharing struct {
	Title   string
	Message string
	Roles   []string
}

// NewSetSimpleSharing returns a fully parsed SetSimpleSharing object
func NewSetSimpleSharing(stepInfo mapof.Any) (SetSimpleSharing, error) {

	return SetSimpleSharing{
		Title:   first.String(stepInfo.GetString("title"), "Sharing Settings"),
		Message: first.String(stepInfo.GetString("message"), "Determine Who Can See This Stream"),
		Roles:   stepInfo.GetSliceOfString("roles"),
	}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step SetSimpleSharing) AmStep() {}
