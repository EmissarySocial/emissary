package step

import "github.com/benpate/rosetta/mapof"

type StripeSetup struct{}

func NewStripeSetup(stepInfo mapof.Any) (StripeSetup, error) {
	return StripeSetup{}, nil
}

func (step StripeSetup) AmStep() {}
