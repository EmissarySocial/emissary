package model

// Transition describes a connection from one state to another
type Transition struct {
	Label       string   `json:"label"`       // Human-friendly label to use for this Transition
	Form        string   `json:"form"`        // ID of the User-facing Form to be filled out in order to complete this Transition
	Permissions []string `json:"permissions"` // List of Permissions required to apply this Transition.
	Actions     []Action `json:"actions"`     // Pipeline of Actions to apply when this Transition is called.
	NextStateID string   `json:"nextStateId"` // ID of the State to set after this Transition is complete
	NextView    string   `json:"nextView"`    // The next view to show after the transition
}
