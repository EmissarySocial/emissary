package step

import "github.com/benpate/rosetta/maps"

type StripeComplete struct{}

func NewStripeComplete(stepInfo maps.Map) (StripeComplete, error) {
	return StripeComplete{}, nil
}

func (step StripeComplete) AmStep() {}
