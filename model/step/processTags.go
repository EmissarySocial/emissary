package step

import (
	"strings"

	"github.com/benpate/rosetta/mapof"
)

// ProcessTags is an action that can add new sub-streams to the domain.
type ProcessTags struct {
	Paths []string
}

// NewProcessTags returns a fully initialized ProcessTags record
func NewProcessTags(stepInfo mapof.Any) (ProcessTags, error) {

	pathString := stepInfo.GetString("paths")
	pathSlice := strings.Split(pathString, ",")

	for index := range pathSlice {
		pathSlice[index] = strings.TrimSpace(pathSlice[index])
	}

	return ProcessTags{
		Paths: pathSlice,
	}, nil
}

// AmStep is here to verify that this struct is a build pipeline step
func (step ProcessTags) AmStep() {}
