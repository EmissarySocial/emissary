package model

import (
	"github.com/benpate/data/journal"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/compare"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product represents a product or subscription that is defined by
// a Merchant Account.  This value is not stored locally, but is passed around
// after being retrieved by the Merchant Account's API.
type Product struct {
	ProductID         primitive.ObjectID `bson:"_id"`               // Unique identifier for this Product
	UserID            primitive.ObjectID `bson:"userId"`            // The User that owns this Product
	MerchantAccountID primitive.ObjectID `bson:"merchantAccountId"` // The Merchant Account where this Product is defined
	RemoteID          string             `bson:"remoteId"`          // The ID of the Product as defined by the Merchant Account
	Name              string             `bson:"name"`              // The name of the Product as defined by the Merchant Account
	Price             string             `bson:"price"`             // The price description of the Product as defined by the Merchant Account
	Icon              string             `bson:"icon"`              // The icon of the Product as defined by the Merchant Account
	AdminHref         string             `bson:"adminHref"`         // URL to the Merchant Account's admin page for this Product

	journal.Journal `bson:",inline"`
}

func NewProduct() Product {
	return Product{
		ProductID: primitive.NewObjectID(),
	}
}

func (product Product) ID() string {
	return product.ProductID.Hex()
}

func (product Product) LookupCode() form.LookupCode {
	return form.LookupCode{
		Group: product.Name,
		Label: product.Price,
		Value: "MA:" + product.MerchantAccountID.Hex() + ":" + product.ProductID.Hex(),
		Icon:  product.Icon,
		Href:  product.AdminHref,
	}
}

func SortProducts(p1 Product, p2 Product) int {

	if comparison := compare.String(p1.Name, p2.Name); comparison != 0 {
		return comparison
	}

	return compare.String(p1.Price, p2.Price)
}
