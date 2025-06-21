package model

import (
	"github.com/benpate/form"
	"github.com/benpate/rosetta/compare"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RemoteProduct represents a product or subscription that is defined by
// a Merchant Account.  This value is not stored locally, but is passed around
// after being retrieved by the Merchant Account's API.
type RemoteProduct struct {
	MerchantAccountID primitive.ObjectID
	ProductID         string
	Name              string
	Description       string
	Icon              string
	AdminHref         string // URL to the Merchant Account's admin page for this product
}

func (product RemoteProduct) Token() string {
	return "MA:" + product.MerchantAccountID.Hex() + ":" + product.ProductID
}

func (product RemoteProduct) LookupCode() form.LookupCode {
	return form.LookupCode{
		Group: product.Name,
		Label: product.Description,
		Value: "MA:" + product.MerchantAccountID.Hex() + ":" + product.ProductID,
		Icon:  product.Icon,
		Href:  product.AdminHref,
	}
}

func SortRemoteProducts(p1 RemoteProduct, p2 RemoteProduct) int {

	if comparison := compare.String(p1.Name, p2.Name); comparison != 0 {
		return comparison
	}

	return compare.String(p1.Description, p2.Description)
}
