package step

import "github.com/benpate/datatype"

type StripeComplete struct{}

func NewStripeComplete(stepInfo datatype.Map) (StripeComplete, error) {
	return StripeComplete{}, nil
}

func (step StripeComplete) AmStep() {}
