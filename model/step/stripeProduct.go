package step

import "github.com/benpate/datatype"

type StripeProduct struct{}

func NewStripeProduct(stepInfo datatype.Map) (StripeProduct, error) {
	return StripeProduct{}, nil
}

func (step StripeProduct) AmStep() {}
