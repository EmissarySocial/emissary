package model

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	Label       string                `json:"label"`       // Human-friendly label to be displayed in lists
	Views       map[string]string     `json:"views"`       // Array of view IDs that can be viewed when a Stream is in this state.
	Transitions map[string]Transition `json:"transitions"` // Array of transitions that can be applied to Streams in this State
}
