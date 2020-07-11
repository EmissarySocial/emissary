package model

// Transition describes a connection from one state to another
type Transition struct {
	ID          string   // Unique Identifier (within this Template) of this Transition
	Label       string   // Human-friendly label to use for this Transition
	StateID     string   // ID of the State to set after this Transition is complete
	Form        Form     // User-facing Form to be filled out in order to complete this Transition
	Actions     []Action // Pipeline of Actions to apply when this Transition is called.
	Permissions []string // List of Permissions required to apply this Transition.
}
