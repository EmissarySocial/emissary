package step

import (
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/mapof"
)

type StripeProduct struct {
	Title string
}

func NewStripeProduct(stepInfo mapof.Any) (StripeProduct, error) {
	return StripeProduct{
		Title: first.String(stepInfo.GetString("title"), "Edit Product"),
	}, nil
}

func (step StripeProduct) AmStep() {}
