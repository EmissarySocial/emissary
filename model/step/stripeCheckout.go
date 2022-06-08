package step

import "github.com/benpate/datatype"

type StripeCheckout struct{}

func NewStripeCheckout(stepInfo datatype.Map) (StripeCheckout, error) {
	return StripeCheckout{}, nil
}

func (step StripeCheckout) AmStep() {}
