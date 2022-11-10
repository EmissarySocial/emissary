package step

import (
	"github.com/benpate/rosetta/maps"
)

// AddSubscription is an action that can add a subscription for the current user
type AddSubscription struct{}

// NewAddSubscription returns a fully initialized AddSubscription record
func NewAddSubscription(stepInfo maps.Map) (AddSubscription, error) {
	return AddSubscription{}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step AddSubscription) AmStep() {}
