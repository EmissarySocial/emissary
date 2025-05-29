package step

import (
	"github.com/benpate/rosetta/mapof"
)

// Sort is a Step that can update multiple records at once
type Sort struct {
	Model   string
	Keys    string
	Values  string
	Message string
}

func NewSort(stepInfo mapof.Any) (Sort, error) {

	return Sort{
		Model:   stepInfo.GetString("model"),
		Keys:    first(stepInfo.GetString("keys"), "_id"),
		Values:  first(stepInfo.GetString("values"), "rank"),
		Message: stepInfo.GetString("message"),
	}, nil
}

// Name returns the name of the step, which is used in debugging.
func (step Sort) Name() string {
	return "set-sort"
}

// RequiredStates returns a slice of states that must be defined any Template that uses this Step
func (step Sort) RequiredStates() []string {
	return []string{}
}

// RequiredRoles returns a slice of roles that must be defined any Template that uses this Step
func (step Sort) RequiredRoles() []string {
	return []string{}
}
