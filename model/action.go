package model

// Action configures an individual action function that will be executed when a stream transitions from one state to another.
type Action struct {
	Function string
}
