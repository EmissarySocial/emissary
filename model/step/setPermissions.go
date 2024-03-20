package step

import (
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
)

// SetRoles represents an action-step that can update the custom data stored in a Stream
type SetRoles struct {
	CopyParent bool
	Roles      map[string][]string
}

// NewSetRoles returns a fully initialized SetRoles object
func NewSetRoles(stepInfo mapof.Any) (SetRoles, error) {

	result := SetRoles{
		Roles: make(map[string][]string),
	}

	for key, value := range stepInfo {
		if key == "do" {
			continue
		}

		if key == "copy-parent" {
			result.CopyParent = convert.Bool(value)
		}

		result.Roles[key] = convert.SliceOfString(value)
	}

	return result, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step SetRoles) AmStep() {}
