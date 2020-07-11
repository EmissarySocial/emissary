package model

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	ID          string       // Unique ID (within this template) of this state
	Label       string       // Human-friendly label to be displayed in lists
	Views       []string     // Array of view IDs that can be viewed when a Stream is in this state.
	Transitions []Transition // Array of transitions that can be applied to Streams in this State
}
