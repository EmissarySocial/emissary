package step

import (
	"github.com/benpate/rosetta/maps"
)

// WebSub represents an action-step that can render a Stream into HTML
type WebSub struct {
}

// NewWebSub generates a fully initialized WebSub step.
func NewWebSub(stepInfo maps.Map) (WebSub, error) {
	return WebSub{}, nil
}

// AmStep is here only to verify that this struct is a render pipeline step
func (step WebSub) AmStep() {}
