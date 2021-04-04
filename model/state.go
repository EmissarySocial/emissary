package model

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	StateID     string       `json:"stateId"      bson:"stateId"`    // Unique ID for this state (within this Template)
	Roles       []string     `json:"roles"       bson:"roles"`       // Array of role names required to access Streams in this State.  This list may be limited further by roles required for specific views.
	Views       []View       `json:"views"       bson:"views"`       // Array of view IDs that can be viewed when a Stream is in this state.
	Transitions []Transition `json:"transitions" bson:"transitions"` // Array of transitions that can be applied to Streams in this State
}

// NewState returns a fully initialized State object.
func NewState() State {
	return State{
		Roles:       make([]string, 0),
		Views:       make([]View, 0),
		Transitions: make([]Transition, 0),
	}
}

// Transition looks up a Transition in this State, using the provided transitionID
// If found, the transition is returned along with TRUE
// If not found, an empty transition is returned along with FALSE
func (s State) Transition(transitionID string) (*Transition, bool) {

	for index := range s.Transitions {
		if s.Transitions[index].TransitionID == transitionID {
			return &(s.Transitions[index]), true
		}
	}

	return nil, false
}

// View searches for the first view in this stream that matches the provided ID.
// If found, the view is returned along with a TRUE.
// If no view matches, and empty view is returned along with a FALSE.
func (s State) View(viewName string) (View, bool) {

	for _, view := range s.Views {
		if view.ViewID == viewName {
			return view, true
		}
	}

	return View{}, false
}

// MatchAnonymous returns TRUE if this state does not require
// any access privileges.
func (s State) MatchAnonymous() bool {
	return len(s.Roles) == 0
}

// MatchRoles returns TRUE if one or more of the provided roles matches the requirements for this State.
// If no roles are defined for this State, then access is always granted.
func (s State) MatchRoles(roles ...string) bool {

	if s.MatchAnonymous() {
		return true
	}

	for i := range roles {
		for j := range s.Roles {
			if roles[i] == s.Roles[j] {
				return true
			}
		}
	}

	return false
}
