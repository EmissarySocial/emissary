package model

import (
	"github.com/benpate/datatype"
)

// Action holds the data for actions that can be performed on any Stream from a particular Template.
type Action struct {
	ActionID string         `json:"actionID" bson:"actionID"` // Unique ID for this action.
	Roles    []string       `json:"roles"    bson:"roles"`    // List of roles required to execute this Action.  If empty, then none are required.
	States   []string       `json:"states"   bson:"states"`   // List of states required to execute this Action.  If empty, then one are required.
	Step     string         `json:"step"     bson:"step"`     // Shortcut for a single step to execute for this Action (all parameters are defaults)
	Steps    []datatype.Map `json:"steps"    bson:"steps"`    // List of steps to execute when GET-ing or POST-ing this Action.
}

// NewAction returns a fully initialized Action
func NewAction() Action {
	return Action{
		Roles:  make([]string, 0),
		States: make([]string, 0),
		Steps:  make([]datatype.Map, 0),
	}
}

func (action Action) IsEmpty() bool {
	return len(action.Steps) == 0
}

// UserCan returns TRUE if this action is permitted on a stream (using the provided authorization)
func (action Action) UserCan(stream *Stream, authorization *Authorization) bool {

	// If present, "States" limits the states where this action can take place
	if len(action.States) > 0 {
		// If states are present, then the current state MUST be included in the list.
		// Otherwise, reject this action.
		if !matchOne(action.States, stream.StateID) {
			return false
		}
	}

	// If present, "Roles" limits the user roles that can take this action
	if len(action.Roles) > 0 {

		// The user must have AT LEAST ONE of the named roles to take this action.
		// If not, reject this action.
		roles := stream.Roles(authorization)

		if !matchAny(roles, action.Roles) {
			return false
		}
	}

	// All filters have passed.  Allow this action.
	return true
}

// Validate runs any required post-processing steps after
// an action is loaded from JSON
func (action *Action) Validate() {

	// Convert single "step" shortcut into a list of actual steps
	if len(action.Steps) == 0 {
		if action.Step != "" {
			// Convert action.Step into a default action
			action.Steps = []datatype.Map{{
				"step": action.Step,
			}}
			action.Step = ""
		}
	}

	// Push actionID into every step.
	for i := range action.Steps {
		action.Steps[i]["actionId"] = action.ActionID
	}

}

// matchOne returns TRUE if the value matches one (or more) of the values in the slice
func matchOne(slice []string, value string) bool {
	for index := range slice {
		if slice[index] == value {
			return true
		}
	}

	return false
}

// matchAny returns TRUE if any of the values in slice1 are equal to any of the values in slice2
func matchAny(slice1 []string, slice2 []string) bool {

	for index1 := range slice1 {
		for index2 := range slice2 {
			if slice1[index1] == slice2[index2] {
				return true
			}
		}
	}

	return false
}
