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

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SetThumbnail) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SetThumbnail) RequiredRoles() []string {
	return []string{}
}
