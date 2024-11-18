package step

import (
	"github.com/benpate/rosetta/mapof"
)

// DeleteArchive represents an action-step that removes a named archive from a Stream
type DeleteArchive struct {
	Token string
}

// NewDeleteArchive returns a fully initialized DeleteArchive object
func NewDeleteArchive(stepInfo mapof.Any) (DeleteArchive, error) {
	return DeleteArchive{
		Token: stepInfo.GetString("token"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step DeleteArchive) AmStep() {}
