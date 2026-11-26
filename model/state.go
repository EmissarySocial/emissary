package model

// State defines an individual state that a Template/Stream can be in.  States are the basis
// for transitions, forms, and actions.
type State struct {
	StateID     string `json:"stateId"     bson:"stateId"`     // Unique ID for this state (within this Template)
	Label       string `json:"label"       bson:"label"`       // Human-friendly label for this State
	Description string `json:"description" bson:"description"` // Description of this State
	IsOutbox    bool   `json:"outbox"      bson:"outbox"`      // TRUE if Streams in this State are sent to their author's followers
}

// NewState returns a fully initialized State object.
func NewState() State {
	return State{}
}
