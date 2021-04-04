package model

import (
	"github.com/benpate/exp"
	"github.com/benpate/form"
)

// Transition describes a connection from one state to another
type Transition struct {
	TransitionID string         `json:"transitionId" bson:"transitionId"`
	Guard        exp.Expression `json:"guard" bson:"guard"` // Guard expression checks if this Transition is legal or not.
	Form         form.Form      `json:"form"`               // ID of the User-facing Form to be filled out in order to complete this Transition
	Roles        []string       `json:"roles"`              // List of Permissions required to apply this Transition.
	Actions      []Action       `json:"actions"`            // Pipeline of Actions to apply when this Transition is called.
	NextState    string         `json:"nextState"`          // ID of the State to set after this Transition is complete
	NextView     string         `json:"nextView"`           // The next view to show after the transition
}

// NewTransition returns a fully populated Transition object
func NewTransition() Transition {

	return Transition{
		Roles:   make([]string, 0),
		Actions: make([]Action, 0),
	}
}

// MatchAnonymous returns TRUE if this Transition requires
// no access permissions.  This does not take into account
// whether or not the Stream CONTAINING this Transition also
// allows anonymous access.
func (t Transition) MatchAnonymous() bool {
	return len(t.Roles) == 0
}

// MatchRoles returns TRUE if one or more of the provided roles matches the requirements for this Transition.
// If no roles are defined for this Transition, then access is always granted.
func (t Transition) MatchRoles(roles ...string) bool {

	if t.MatchAnonymous() {
		return true
	}

	for i := range roles {
		for j := range t.Roles {
			if roles[i] == t.Roles[j] {
				return true
			}
		}
	}

	return false
}
