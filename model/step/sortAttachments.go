package step

import (
	"github.com/benpate/rosetta/mapof"
)

// SortAttachments is a Step that can update multiple records at once
type SortAttachments struct {
	Keys    string
	Values  string
	Message string
}

func NewSortAttachments(stepInfo mapof.Any) (SortAttachments, error) {

	return SortAttachments{
		Keys:    first(stepInfo.GetString("keys"), "_id"),
		Values:  first(stepInfo.GetString("values"), "rank"),
		Message: stepInfo.GetString("message"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step SortAttachments) Name() string {
	return "sort-attachments"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step SortAttachments) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step SortAttachments) RequiredRoles() []string {
	return []string{}
}
