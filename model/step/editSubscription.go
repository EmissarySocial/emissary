package step

import (
	"github.com/benpate/rosetta/maps"
)

// EditSubscription is an action that can update subscription details for the current user
type EditSubscription struct{}

// NewEditSubscription returns a fully initialized EditSubscription record
func NewEditSubscription(stepInfo maps.Map) (EditSubscription, error) {
	return EditSubscription{}, nil
}

// AmStep is here to verify that this struct is a render pipeline step
func (step EditSubscription) AmStep() {}
