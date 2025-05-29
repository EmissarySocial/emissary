package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SetPrivileges represents an action that can edit a top-level folder in the Domain
type SetPrivileges struct {
	Title string
}

// NewSetPrivileges returns a fully parsed SetPrivileges object
func NewSetPrivileges(stepInfo mapof.Any) (SetPrivileges, error) {

	return SetPrivileges{
		Title: first(stepInfo.GetString("title"), "Product Settings"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetPrivileges) AmStep() {}
