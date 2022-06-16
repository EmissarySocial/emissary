package step

import (
	"github.com/benpate/datatype"
	"github.com/benpate/first"
)

type StripeProduct struct {
	Title string
}

func NewStripeProduct(stepInfo datatype.Map) (StripeProduct, error) {
	return StripeProduct{
		Title: first.String(stepInfo.GetString("title"), "Edit Product"),
	}, nil
}

func (step StripeProduct) AmStep() {}
