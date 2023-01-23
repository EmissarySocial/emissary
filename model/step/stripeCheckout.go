package step

import "github.com/benpate/rosetta/mapof"

type StripeCheckout struct{}

func NewStripeCheckout(stepInfo mapof.Any) (StripeCheckout, error) {
	return StripeCheckout{}, nil
}

func (step StripeCheckout) AmStep() {}
