package step

import "github.com/benpate/rosetta/mapof"

type StripeComplete struct{}

func NewStripeComplete(stepInfo mapof.Any) (StripeComplete, error) {
	return StripeComplete{}, nil
}

func (step StripeComplete) AmStep() {}
