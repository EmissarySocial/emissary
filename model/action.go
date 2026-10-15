package model

import (
	"html/template"
	texttemplate "text/template"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
)

// Action holds the data for actions that can be performed on any Stream from a particular Template.
type Action struct {
	Roles      sliceof.String               `json:"roles"      bson:"roles"`      // List of roles required to execute this Action.  If empty, then none are required.
	States     sliceof.String               `json:"states"     bson:"states"`     // List of states required to execute this Action.  If empty, then none are required.
	StateRoles mapof.Object[sliceof.String] `json:"stateRoles" bson:"stateRoles"` // Map of states -> list of roles that can perform this Action (when a stream is in the given state)
	Steps      sliceof.Object[step.Step]    `json:"steps"      bson:"steps"`      // List of steps to execute when GET-ing or POST-ing this Action.
	AccessList mapof.Object[sliceof.String] `json:"-"          bson:"-"`          // Map of states -> set of roles that can perform this Action.
}

// NewAction returns a fully initialized Action
func NewAction() Action {
	return Action{
		Roles:      sliceof.NewString(),
		States:     sliceof.NewString(),
		StateRoles: mapof.NewObject[sliceof.String](),
		Steps:      sliceof.NewObject[step.Step](),
		AccessList: mapof.NewObject[sliceof.String](),
	}
}

// CalcAccessList translates the roles, states, and stateRoles settings into a compact AccessList that
// can quickly determine if a user can perform this action on objects given their current state.
func (action *Action) CalcAccessList(template *Template) {

	// Initialize/Reset the AccessList
	action.AccessList = mapof.NewObject[sliceof.String]()

	// Calculate an AccessList for each state defined in the Template
	for stateID := range template.States {

		// If specific states are required to perform this action, then verify that this state...
		if len(action.States) > 0 {

			// If the current state is not allowed, this action cannot be performed.
			// Skipping means that a zero accessList (no permissions) will be returned for this state.
			if action.States.NotContains(stateID) {
				continue
			}
		}

		// Set the AccessList for this State
		action.AccessList[stateID] = action.calcAccessListForStateAndRole(stateID)
	}
}

// calcAccessListForStateAndRole returns every role allowed to run this Action while a Stream is in the provided state
func (action *Action) calcAccessListForStateAndRole(stateID string) sliceof.String {

	// Create an AccessList for Streams in this State
	result := sliceof.NewString()
	allowedRoles := append(action.Roles, action.StateRoles[stateID]...)

	// Nilaway guard
	if allowedRoles == nil {
		allowedRoles = sliceof.NewString()
	}

	// Special case for Anonymous users overrides all other roles
	if allowedRoles.Contains(MagicRoleAnonymous) {
		result = []string{MagicRoleAnonymous}
		return result
	}

	// Special case for Authenticated users overrides all other roles
	if allowedRoles.Contains(MagicRoleAuthenticated) {
		result = []string{MagicRoleAuthenticated}
		return result
	}

	// Calculate the roles in the AccessList
	for _, roleID := range allowedRoles {

		switch roleID {

		// MagicRoleOwner represents the domain owner who can do anything.
		// No flag is required here because domain owners can already do everything.
		case MagicRoleOwner:

		// MagicRoleMyself allows Users to perform actions on their own profiles
		case MagicRoleMyself:
			result = append(result, MagicRoleMyself)

		// MagicRoleAuthor allows Users to perform actions on Streams that they created
		case MagicRoleAuthor:
			result = append(result, MagicRoleAuthor)

		// All other privileges are granted via membership in a group or purchase of a product
		default:
			result = append(result, roleID)
		}
	}

	// Unique-ify the lists of group and product roles
	result = slice.Unique(result)

	return result
}

// AllowedRoles returns the roles allowed to perform this Action on an object in the provided state,
// including the magic roles "anonymous", "authenticated", "self", and "author" (but never "owner")
func (action *Action) AllowedRoles(stateID string) sliceof.String {
	return action.AccessList[stateID]
}

// Debug is a debugging method that outputs all of the contents of an Action
// without displaying steps/templates (which are huge)
func (action *Action) Debug() mapof.Any {

	return mapof.Any{
		"roles":      action.Roles,
		"states":     action.States,
		"stateRoles": action.StateRoles,
		"accessList": action.AccessList,
		"steps": slice.Map(action.Steps, func(step step.Step) string {
			return step.Name()
		}),
	}
}

/******************************************
 * Marshalling Methods
 ******************************************/

// UnmarshalMap unmarshals the provided map into this Action object, parsing every
// step template with funcMap
func (action *Action) UnmarshalMap(data map[string]any, funcMap template.FuncMap) error {

	const location = "model.Action.UnmarshalMap"

	// Import easy values
	action.Roles = convert.SliceOfString(data["roles"])
	action.States = convert.SliceOfString(data["states"])

	// Import stateRoles
	action.StateRoles = make(mapof.Object[sliceof.String])
	stateRoles := convert.MapOfAny(data["stateRoles"])
	for key, value := range stateRoles {
		action.StateRoles[key] = convert.SliceOfString(value)
	}

	// Import steps
	stepsInfo := convert.SliceOfMap(data["steps"])
	if pipeline, err := step.NewPipeline(stepsInfo, texttemplate.FuncMap(funcMap)); err == nil {
		action.Steps = pipeline
	} else {
		return derp.Wrap(err, location, "Reading steps", stepsInfo)
	}

	// intentionally ignoring validation errors here
	// so that we can generate more useful error messages later.
	// for now, everything is valid. everything is fine. nothing to see here. move along, please.

	// NOT VALIDATING EMPTY ACTIONS
	// NOT VALIDATING INCORRECT ROLES
	// NOT VALIDATING INCORRECT STATES

	return nil
}

// unmarshalActions adds each action in data to actions, parsing every step template with funcMap
func unmarshalActions(actions mapof.Object[Action], data map[string]map[string]any, funcMap template.FuncMap) error {

	for actionID, actionData := range data {

		action := NewAction()

		if err := action.UnmarshalMap(actionData, funcMap); err != nil {
			return derp.Wrap(err, "model.unmarshalActions", "Invalid action", actionID)
		}

		actions[actionID] = action
	}

	// Lights, camera, actions
	return nil
}
