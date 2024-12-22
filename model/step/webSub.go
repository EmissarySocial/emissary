package step

import "github.com/benpate/rosetta/mapof"

// WebSub is a Step that can build a Stream into HTML
type WebSub struct {
}

// NewWebSub generates a fully initialized WebSub step.
func NewWebSub(stepInfo mapof.Any) (WebSub, error) {
	return WebSub{}, nil
}

// AmStep is here only to verify that this struct is a build pipeline step
func (step WebSub) AmStep() {}
