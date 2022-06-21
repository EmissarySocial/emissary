package step

import "github.com/benpate/rosetta/maps"

type StripeSetup struct{}

func NewStripeSetup(stepInfo maps.Map) (StripeSetup, error) {
	return StripeSetup{}, nil
}

func (step StripeSetup) AmStep() {}
