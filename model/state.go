package model

import "github.com/davecgh/go-spew/spew"

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	StateID     string       `json:"stateId"      bson:"stateId"`    // Unique ID for this state (within this Template)
	Views       []View       `json:"views"       bson:"views"`       // Array of view IDs that can be viewed when a Stream is in this state.
	Transitions []Transition `json:"transitions" bson:"transitions"` // Array of transitions that can be applied to Streams in this State
}

// NewState returns a fully initialized State object.
func NewState() State {
	return State{
		Views:       make([]View, 0),
		Transitions: make([]Transition, 0),
	}
}

// Transition looks up a Transition in this State, using the provided transitionID
// If found, the transition is returned along with TRUE
// If not found, an empty transition is returned along with FALSE
func (s State) Transition(transitionID string) (*Transition, bool) {

	if s.Transitions == nil {
		return nil, false
	}

	for _, transition := range s.Transitions {
		if transition.TransitionID == transitionID {
			return &transition, true
		}
	}

	return nil, false
}

// View searches for the first view in this stream that matches the provided ID.
// If found, the view is returned along with a TRUE.
// If no view matches, and empty view is returned along with a FALSE.
func (s State) View(viewName string) (*View, bool) {

	spew.Dump("Searching for View: " + viewName)

	if s.Views == nil {
		spew.Dump("NIL")
		return nil, false
	}

	for _, view := range s.Views {
		spew.Dump("checking: " + view.ViewID)
		if view.ViewID == viewName {
			return &view, true
		}
	}

	spew.Dump("NO VIEW FOUND")
	return nil, false
}
