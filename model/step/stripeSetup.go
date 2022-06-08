package step

import "github.com/benpate/datatype"

type StripeSetup struct{}

func NewStripeSetup(stepInfo datatype.Map) (StripeSetup, error) {
	return StripeSetup{}, nil
}

func (step StripeSetup) AmStep() {}
