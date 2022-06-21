package step

import "github.com/benpate/rosetta/maps"

type StripeCheckout struct{}

func NewStripeCheckout(stepInfo maps.Map) (StripeCheckout, error) {
	return StripeCheckout{}, nil
}

func (step StripeCheckout) AmStep() {}
