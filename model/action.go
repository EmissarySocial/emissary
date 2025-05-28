package model

import (
	"strings"

	"github.com/EmissarySocial/emissary/model/step"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/convert"
	"github.com/benpate/rosetta/mapof"
	"github.com/benpate/rosetta/slice"
	"github.com/benpate/rosetta/sliceof"
	"github.com/hjson/hjson-go/v4"
)

// Action holds the data for actions that can be performed on any Stream from a particular Template.
type Action struct {
	Roles      sliceof.String               `json:"roles"      bson:"roles"`      // List of roles required to execute this Action.  If empty, then none are required.
	States     sliceof.String               `json:"states"     bson:"states"`     // List of states required to execute this Action.  If empty, then one are required.
	StateRoles mapof.Object[sliceof.String] `json:"stateRoles" bson:"stateRoles"` // Map of states -> list of roles that can perform this Action (when a stream is in the given state)
	Steps      sliceof.Object[step.Step]    `json:"steps"      bson:"steps"`      // List of steps to execute when GET-ing or POST-ing this Action.
	AccessList mapof.Object[AccessList]     `json:"-"          bson:"-"`          // Map of states -> set of roles that can perform this Action.
}

// NewAction returns a fully initialized Action
func NewAction() Action {
	return Action{
		Roles:      sliceof.NewString(),
		States:     sliceof.NewString(),
		StateRoles: mapof.NewObject[sliceof.String](),
		Steps:      sliceof.NewObject[step.Step](),
		AccessList: mapof.NewObject[AccessList](),
	}
}

// CalcAccessList translates the roles, states, and stateRoles settings into a compact AccessList that
// can quickly determine if a user can perform this action on objects given their current state.
func (action *Action) CalcAccessList(template *Template) error {

	const location = "model.Action.CalcAccessList"

	// Initialize/Reset the AccessList
	action.AccessList = mapof.NewObject[AccessList]()

	// Verify that all Roles exist in the list of templateRoles
	for _, role := range action.Roles {
		if !template.IsValidRole(role) {
			return derp.InternalError(
				location,
				"Undefined role used in Action.Roles.  Roles must be defined in the Template before use.",
				"template: "+template.TemplateID,
				"available roles: "+strings.Join(template.AccessRoles.Keys(), ", "),
				"selected role: "+role,
			)
		}
	}

	// Verify that all States and Roles exist in the list of templateStates
	for stateID, roles := range action.StateRoles {

		if !template.IsValidState(stateID) {
			return derp.InternalError(
				location,
				"Undefined state used in StateRoles. States must be defined in the Template before use.",
				"template: "+template.TemplateID,
				"available states: "+strings.Join(template.States.Keys(), ", "),
				"selected state: "+stateID,
			)
		}
		for _, role := range roles {
			if !template.IsValidRole(role) {
				return derp.InternalError(
					location,
					"Undefined role used in Action.StateRoles.  Roles must be defined in the Template before use.",
					"template: "+template.TemplateID,
					"state: "+stateID,
					"available roles: "+strings.Join(template.AccessRoles.Keys(), ", "),
					"selected role: "+role,
				)
			}
		}
	}

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
		action.AccessList[stateID] = action.calcAccessListForStateAndRole(template, stateID)
	}

	return nil
}

func (action *Action) calcAccessListForStateAndRole(template *Template, stateID string) AccessList {

	// Create an AccessList for Streams in this State
	result := NewAccessList()
	allowedRoles := append(action.Roles, action.StateRoles[stateID]...)

	if allowedRoles.Contains(MagicRoleAnonymous) {
		result.Anonymous = true
		return result
	}

	// Special case for Anonymous users overrides all other roles
	if allowedRoles.Contains(MagicRoleAnonymous) {
		result.Anonymous = true
		return result
	}

	// Special case for Authenticated users overrides all other roles
	if allowedRoles.Contains(MagicRoleAuthenticated) {
		result.Authenticated = true
		return result
	}

	// Calculate the roles in the AccessList
	for _, roleID := range allowedRoles {

		switch roleID {

		// MagicRoleOwner represents the domain owner who can do anything.
		// No flag is required here because domain owners can already do everything.
		case MagicRoleOwner:

		// MagicRoleMyself allows Users to perform actions on their own profies
		case MagicRoleMyself:
			result.Self = true

		// MagicRoleAuthor allows Users to perform actions on Streams that they created
		case MagicRoleAuthor:
			result.Author = true

		// All other privileges are granted via membership in a group or purchase of a product
		default:
			role := template.AccessRoles[roleID] // save becuase this was already checked above

			if role.Purchasable {
				result.Privileges = append(result.Privileges, roleID)
			} else {
				result.Groups = append(result.Groups, roleID)
			}
		}
	}

	// Unique-ify the lists of group and product roles
	result.Groups = slice.Unique(result.Groups)
	result.Privileges = slice.Unique(result.Privileges)

	return result
}

// AllowedRoles returns a slice of roles that are allowed to perform this action,
// based on the state of the object.  This list includes
// system roles like "anonymous", "authenticated", "self", "author", and "owner".
func (action *Action) AllowedRoles(stateID string) []string {
	return action.AccessList[stateID].Roles()
}

/******************************************
 * Marshalling Methods
 ******************************************/

// UnmarshalJSON unmarshals the JSON data into this Action object.
func (action *Action) UnmarshalJSON(data []byte) error {
	var asMap map[string]any

	if err := hjson.Unmarshal(data, &asMap); err != nil {
		return derp.Wrap(err, "model.Action.UnmarshalJSON", "Invalid JSON")
	}

	return action.UnmarshalMap(asMap)
}

// UnmarshalMap unmarshals the provided map into this Action object.
func (action *Action) UnmarshalMap(data map[string]any) error {

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
	if pipeline, err := step.NewPipeline(stepsInfo); err == nil {
		action.Steps = pipeline
	} else {
		return derp.Wrap(err, location, "Error reading steps", stepsInfo)
	}

	// intentionally ignoring validation errors here
	// so that we can generate more useful error messages later.
	// for now, everything is valid. everything is fine. nothing to see here. move along, please.

	// NOT VALIDATING EMPTY ACTIONS
	// NOT VALIDATING INCORRECT ROLES
	// NOT VALIDATING INCORRECT STATES

	return nil
}
