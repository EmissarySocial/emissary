package step

import (
	"github.com/benpate/rosetta/mapof"
)

// DeleteArchive is a Step that removes a named archive from a Stream
type DeleteArchive struct {
	Token string
}

// NewDeleteArchive returns a fully initialized DeleteArchive object
func NewDeleteArchive(stepInfo mapof.Any) (DeleteArchive, error) {
	return DeleteArchive{
		Token: stepInfo.GetString("token"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step DeleteArchive) Name() string {
	return "delete-archive"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step DeleteArchive) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step DeleteArchive) RequiredRoles() []string {
	return []string{}
}
