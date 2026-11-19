package step

import "github.com/benpate/rosetta/mapof"

// SetThumbnail is a Step that writes the object's first image attachment into a schema path
type SetThumbnail struct {
	Path     string
	Category string
}

// NewSetThumbnail returns a fully initialized SetThumbnail step, or an error if its configuration is invalid
func NewSetThumbnail(stepInfo mapof.Any) (SetThumbnail, error) {
	return SetThumbnail{
		Path:     stepInfo.GetString("path"),
		Category: stepInfo.GetString("category"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SetThumbnail) Name() string {
	return "set-thumbnail"
}

// RequiredModel returns the name of the model object that MUST be present in the Template.
// If this value is not empty, then the Template MUST use this model object.
func (step SetThumbnail) RequiredModel() string {
	return ""
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetThumbnail) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetThumbnail) RequiredRoles() []string {
	return []string{}
}
