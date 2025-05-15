package model

import (
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
	Roles      sliceof.String                `json:"roles"      bson:"roles"`      // List of roles required to execute this Action.  If empty, then none are required.
	States     sliceof.String                `json:"states"     bson:"states"`     // List of states required to execute this Action.  If empty, then one are required.
	StateRoles mapof.Object[sliceof.String]  `json:"stateRoles" bson:"stateRoles"` // Map of states -> list of roles that can perform this Action (when a stream is in the given state)
	Steps      sliceof.Object[step.Step]     `json:"steps"      bson:"steps"`      // List of steps to execute when GET-ing or POST-ing this Action.
	AllowList  mapof.Object[ActionAllowList] `json:"-"          bson:"-"`          // Map of states -> set of users that can perform this Action.
}

// NewAction returns a fully initialized Action
func NewAction() Action {
	return Action{
		Roles:      make(sliceof.String, 0),
		States:     make(sliceof.String, 0),
		StateRoles: make(mapof.Object[sliceof.String]),
		Steps:      make(sliceof.Object[step.Step], 0),
		AllowList:  make(mapof.Object[ActionAllowList]),
	}
}

// CalcAllowList translates the roles, states, and stateRoles settings into a compact AllowList that
// can quickly determine if a user can perform this action on objects given their current state.
func (action *Action) CalcAllowList(template *Template) error {

	const location = "model.Action.CalcAllowList"

	// RULE: Require at leas one Role.
	if len(action.Roles) == 0 {
		return derp.InternalError(location, "Action must have at least one Role.  If none, then use 'owner'")
	}

	// Initialize/Reset the AllowList
	action.AllowList = make(mapof.Object[ActionAllowList])

	// Verify that all Roles exist in the list of templateRoles
	for _, role := range action.Roles {
		if !template.IsValidRole(role) {
			return derp.InternalError(
				location,
				"Invalid role used in Action.Roles.  Roles must be defined in the Template to be used in AllowLists",
				template.TemplateID,
				role,
			)
		}
	}

	// Verify that all States and Roles exist in the list of templateStates
	for stateID, roles := range action.StateRoles {

		if !template.IsValidState(stateID) {
			return derp.InternalError(
				location,
				"Invalid state used in StateRoles. States must be defined in the Template to be used in AllowLists",
				template.TemplateID,
				stateID,
			)
		}
		for _, role := range roles {
			if !template.IsValidRole(role) {
				return derp.InternalError(
					location,
					"Invalid role used in Action.StateRoles.  Roles must be defined in the Template to be used in AllowLists",
					template.TemplateID,
					role,
				)
			}
		}
	}

	// Calculate an AllowList for each state defined in the Template
	for stateID := range template.States {

		// If specific states are required to perform this action, then verify that this state...
		if len(action.States) > 0 {

			// If the current state is not allowed, this action cannot be performed.
			// Skipping means that a zero allowList (no permissions) will be returned for this state.
			if action.States.NotContains(stateID) {
				continue
			}
		}

		// Create an AllowList for Streams in this State
		allowList := NewActionAllowList()
		allowRoles := append(action.Roles, action.StateRoles[stateID]...)

		// Calculate the roles in the AllowList
		for _, roleID := range allowRoles {

			switch roleID {

			// MagicRoleOwner represents the domain owner who can do anything.
			// No flag is required here because domain owners have access to everything.
			case MagicRoleOwner:

			// MagicRoleAnonymous is a shortcut for allowing anonymous access
			case MagicRoleAnonymous:
				allowList.Anonymous = true

			// MagicRoleAuthenticated allows access to any identified user
			case MagicRoleAuthenticated:
				allowList.Authenticated = true

			// MagicRoleMyself allows Users to perform actions on their own profies
			case MagicRoleMyself:
				allowList.Self = true

			// MagicRoleAuthor allows Users to perform actions on Streams that they created
			case MagicRoleAuthor:
				allowList.Author = true

			// All other privileges are granted via membership in a group or purchase of a product
			default:
				role := template.AccessRoles[roleID] // save becuase this was already checked above

				if role.Purchasable {
					allowList.ProductRoles = append(allowList.ProductRoles, roleID)
				} else {
					allowList.GroupRoles = append(allowList.GroupRoles, roleID)
				}
			}
		}

		// Unique-ifly the lists of group and product roles
		allowList.GroupRoles = slice.Unique(allowList.GroupRoles)
		allowList.ProductRoles = slice.Unique(allowList.ProductRoles)

		// Put the updated allowList back into the map
		action.AllowList[stateID] = allowList
	}

	return nil
}

// AllowedRoles returns a slice of roles that are allowed to perform this action,
// based on the state of the object.  This list includes
// system roles like "anonymous", "authenticated", "self", "author", and "owner".
func (action *Action) AllowedRoles(stateID string) []string {
	return action.AllowList[stateID].Roles()
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
		return derp.Wrap(err, "model.action.UnmarshalMap", "Error reading steps", stepsInfo)
	}

	// If no steps configued, then try the "do" alias
	if len(action.Steps) == 0 {
		if name := convert.String(data["do"]); name != "" {
			action.Steps, _ = step.NewPipeline([]mapof.Any{data})
		}
	}

	return nil
}
