package model

import (
	"encoding/json"

	"github.com/benpate/convert"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/whisperverse/whisperverse/model/step"
)

// Action holds the data for actions that can be performed on any Stream from a particular Template.
type Action struct {
	Roles      []string            `path:"roles"      json:"roles"      bson:"roles"`      // List of roles required to execute this Action.  If empty, then none are required.
	States     []string            `path:"states"     json:"states"     bson:"states"`     // List of states required to execute this Action.  If empty, then one are required.
	RoleStates map[string][]string `path:"roleStates" json:"roleStates" bson:"roleStates"` // Map of roles -> list of states that grant access to this Action.
	Steps      []step.Step         `path:"steps"      json:"steps"      bson:"steps"`      // List of steps to execute when GET-ing or POST-ing this Action.
}

// NewAction returns a fully initialized Action
func NewAction() Action {
	return Action{
		Roles:      make([]string, 0),
		States:     make([]string, 0),
		RoleStates: make(map[string][]string),
		Steps:      make([]step.Step, 0),
	}
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

	// If present, "Roles" and "RoleStates" limit the user roles that can take this action
	if (len(action.Roles) > 0) || (len(action.RoleStates) > 0) {

		// The user must have AT LEAST ONE of the named roles to take this action.
		// If not, reject this action.
		roles := stream.Roles(authorization)

		// If the user matches any of the designated roles, then they can take this action.
		if matchAny(roles, action.Roles) {
			return true
		}

		// Check Roles/States for any limited roles
		for _, role := range roles {

			// If this role is granted limited permissions
			if stateList, ok := action.RoleStates[role]; ok {
				// then check to see if the stream is in a valid state
				// for this limited role to perform this action...
				if matchOne(stateList, stream.StateID) {
					return true
				}
			}
		}

		// Fall through means that there are role-based permissions,
		// but the user does not meet any of them.
		return false
	}

	// All filters have passed.  Allow this action.
	return true
}

func (action *Action) UnmarshalJSON(data []byte) error {
	var asMap map[string]any

	if err := json.Unmarshal(data, &asMap); err != nil {
		return derp.Wrap(err, "model.Action.UnmarshalJSON", "Invalid JSON")
	}

	return action.UnmarshalMap(asMap)
}

func (action *Action) UnmarshalMap(data map[string]any) error {

	// Import easy values
	action.Roles = convert.SliceOfString(data["roles"])
	action.States = convert.SliceOfString(data["states"])

	// Import roleStates
	action.RoleStates = make(map[string][]string, 0)
	roleStates := convert.MapOfInterface(data["roleStates"])
	for key, value := range roleStates {
		action.RoleStates[key] = convert.SliceOfString(value)
	}

	// Import steps
	stepsInfo := convert.SliceOfMap(data["steps"])
	if pipeline, err := step.NewPipeline(stepsInfo); err == nil {
		action.Steps = pipeline
	} else {
		return derp.Wrap(err, "model.action.UnmarshalMap", "Error reading steps", stepsInfo)
	}

	// If no steps configued, then try the "step" alias
	if len(action.Steps) == 0 {
		if name := convert.String(data["step"]); name != "" {
			action.Steps, _ = step.NewPipeline([]datatype.Map{{"step": name}})
		}
	}

	return nil
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
