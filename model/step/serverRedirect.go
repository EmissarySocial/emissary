package step

import (
	"github.com/benpate/rosetta/mapof"
)

// ServerRedirect represents an action-step that forwards the user to a new page.
type ServerRedirect struct {
	On     string
	Action string
}

// NewServerRedirect returns a fully initialized ServerRedirect object
func NewServerRedirect(stepInfo mapof.Any) (ServerRedirect, error) {

	return ServerRedirect{
		On:     first(stepInfo.GetString("on"), "post"),
		Action: stepInfo.GetString("action"),
	}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step ServerRedirect) AmStep() {}
