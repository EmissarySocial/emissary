package step

import "github.com/benpate/rosetta/mapof"

// SetThumbnail is a Step that can update the data.DataMap custom data stored in a Stream
type SetThumbnail struct {
	Path string
}

func NewSetThumbnail(stepInfo mapof.Any) (SetThumbnail, error) {
	return SetThumbnail{
		Path: stepInfo.GetString("path"),
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
