package model

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	StateID     string `path:"stateId"     json:"stateId"     bson:"stateId"`     // Unique ID for this state (within this Template)
	Label       string `path:"label"       json:"label"       bson:"label"`       // Human-friendly label for this State
	Description string `path:"description" json:"description" bson:"description"` // Description of this State
}

// NewState returns a fully initialized State object.
func NewState() State {
	return State{}
}
