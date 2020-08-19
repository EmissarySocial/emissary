package model

// Transition describes a connection from one state to another
type Transition struct {
	ID          string   `json:"id"`          // Unique Identifier (within this Template) of this Transition
	Label       string   `json:"label"`       // Human-friendly label to use for this Transition
	StateID     string   `json:"stateId"`     // ID of the State to set after this Transition is complete
	Form        string   `json:"form"`        // ID of the User-facing Form to be filled out in order to complete this Transition
	Actions     []Action `json:"actions"`     // Pipeline of Actions to apply when this Transition is called.
	Permissions []string `json:"permissions"` // List of Permissions required to apply this Transition.
	NextView    string   `json:"nextView"`    // The next view to show after the transition
}
