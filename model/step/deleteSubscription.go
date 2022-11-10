package step

import (
	"github.com/benpate/rosetta/maps"
)

// DeleteSubscription is an action that can delete a subscription for the current user.
type DeleteSubscription struct{}

// NewDeleteSubscription returns a fully initialized DeleteSubscription record
func NewDeleteSubscription(stepInfo maps.Map) (DeleteSubscription, error) {
	return DeleteSubscription{}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step DeleteSubscription) AmStep() {}
