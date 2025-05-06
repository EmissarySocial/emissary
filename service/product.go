package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product defines a service that manages all content products created and imported by Users.
type Product struct {
	merchantAccountService *MerchantAccount
	collection             data.Collection
}

// NewProduct returns a fully initialized Product service
func NewProduct() Product {
	return Product{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Product) Refresh(collection data.Collection, merchantAccountService *MerchantAccount) {
	service.collection = collection
	service.merchantAccountService = merchantAccountService
}

// Close stops any background processes controlled by this service
func (service *Product) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Product) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Products that match the provided criteria
func (service *Product) Query(criteria exp.Expression, options ...option.Option) ([]model.Product, error) {
	result := make([]model.Product, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Products that match the provided criteria
func (service *Product) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Product records that match the provided criteria
func (service *Product) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Product], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Product.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewProduct), nil
}

// Load retrieves an Product from the database
func (service *Product) Load(criteria exp.Expression, product *model.Product) error {

	if err := service.collection.Load(notDeleted(criteria), product); err != nil {
		return derp.Wrap(err, "service.Product.Load", "Error loading Product", criteria)
	}

	return nil
}

// Save adds/updates an Product in the database
func (service *Product) Save(product *model.Product, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(product); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error validating Product", product)
	}

	// Validate the Merchant Account
	merchantAccount := model.NewMerchantAccount()
	if err := service.merchantAccountService.LoadByUserAndID(product.UserID, product.MerchantAccountID, &merchantAccount); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error loading Merchant Account", product.MerchantAccountID)
	}

	if err := service.merchantAccountService.RefreshAPIKeys(&merchantAccount); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error validating Merchant Account", product.MerchantAccountID)
	}

	product.MerchantAccountType = merchantAccount.Type

	if err := service.merchantAccountService.RefreshProduct(&merchantAccount, product); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error validating Product", product)
	}

	// Save the product to the database
	if err := service.collection.Save(product, note); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error saving Product", product, note)
	}

	return nil
}

// Delete removes an Product from the database (virtual delete)
func (service *Product) Delete(product *model.Product, note string) error {

	// Delete this Product
	if err := service.collection.Delete(product, note); err != nil {
		return derp.Wrap(err, "service.Product.Delete", "Error deleting Product", product, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Product) ObjectType() string {
	return "Product"
}

// New returns a fully initialized model.Product as a data.Object.
func (service *Product) ObjectNew() data.Object {
	result := model.NewProduct()
	return &result
}

func (service *Product) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Product); ok {
		return mention.ProductID
	}

	return primitive.NilObjectID
}

func (service *Product) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Product) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewProduct()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Product) ObjectSave(object data.Object, comment string) error {
	if product, ok := object.(*model.Product); ok {
		return service.Save(product, comment)
	}
	return derp.InternalError("service.Product.ObjectSave", "Invalid Object Type", object)
}

func (service *Product) ObjectDelete(object data.Object, comment string) error {
	if product, ok := object.(*model.Product); ok {
		return service.Delete(product, comment)
	}
	return derp.InternalError("service.Product.ObjectDelete", "Invalid Object Type", object)
}

func (service *Product) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Product", "Not Authorized")
}

func (service *Product) Schema() schema.Schema {
	return schema.New(model.ProductSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Product) QueryByID(userID primitive.ObjectID, productID ...primitive.ObjectID) ([]model.Product, error) {

	criteria := exp.Equal("userId", userID).AndIn("_id", productID)

	products, err := service.Query(criteria)
	if err != nil {
		return nil, derp.Wrap(err, "service.Product.QueryByID", "Error querying Products", criteria)
	}

	return products, nil
}

func (service *Product) QueryAsLookupCodes(userID primitive.ObjectID) ([]form.LookupCode, error) {

	// Query Product for this User
	criteria := exp.Equal("userId", userID)
	products, err := service.Query(criteria)

	if err != nil {
		return nil, derp.Wrap(err, "service.Product.QueryAsLookupCodes", "Error querying Products", criteria)
	}

	// Map the Products into LookupCodes
	result := make([]form.LookupCode, 0)

	for _, product := range products {
		result = append(result, product.LookupCode())
	}

	return result, nil

}

func (service *Product) LoadByID(productID primitive.ObjectID, product *model.Product) error {

	criteria := exp.Equal("_id", productID)

	return service.Load(criteria, product)
}

func (service *Product) LoadByUserAndID(userID primitive.ObjectID, productID primitive.ObjectID, product *model.Product) error {

	criteria := exp.Equal("_id", productID).
		AndEqual("userId", userID)

	return service.Load(criteria, product)
}

func (service *Product) LoadByUserAndToken(userID primitive.ObjectID, token string, product *model.Product) error {

	productID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Product.LoadByToken", "Invalid Token", token)
	}

	return service.LoadByUserAndID(userID, productID, product)
}

// LoadByRemoteID retrieves a single Product using the ID provided by the MerchantAccount
func (service *Product) LoadByRemoteID(remoteID string, product *model.Product) error {

	criteria := exp.Equal("remoteId", remoteID)

	if err := service.Load(criteria, product); err != nil {
		return derp.Wrap(err, "service.Product.LoadByRemoteID", "Error loading Product", criteria)
	}

	return nil
}
