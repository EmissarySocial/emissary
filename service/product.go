package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Product manages all interactions with the Product collection
type Product struct {
	collection             data.Collection
	merchantAccountService *MerchantAccount
}

// NewProduct returns a fully populated Product service
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

}

/******************************************
 * Common Data Methods
 ******************************************/

// Count returns the number of records that match the provided criteria
func (service *Product) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

func (service *Product) Query(criteria exp.Expression, options ...option.Option) ([]model.Product, error) {
	result := make([]model.Product, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Products who match the provided criteria
func (service *Product) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Product from the database
func (service *Product) Load(criteria exp.Expression, result *model.Product) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
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

	// Save the value to the database
	if err := service.collection.Save(product, note); err != nil {
		return derp.Wrap(err, "service.Product.Save", "Error saving Product", product, note)
	}

	return nil
}

// Delete removes an Product from the database (virtual delete)
func (service *Product) Delete(product *model.Product, note string) error {

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

	if product, ok := object.(*model.Product); ok {
		return product.ProductID
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

// LoadByID loads a single model.Product object that matches the provided productID
func (service *Product) LoadByID(productID primitive.ObjectID, result *model.Product) error {
	criteria := exp.Equal("_id", productID)
	return service.Load(criteria, result)
}

func (service *Product) LoadByToken(token string, result *model.Product) error {

	productID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Product.LoadByToken", "Invalid Product ID", token)
	}

	return service.LoadByID(productID, result)
}

// QueryByUser returns a slice of Products that match the provided productIDs
func (service *Product) QueryByUser(userID primitive.ObjectID) (sliceof.Object[model.Product], error) {

	criteria := exp.Equal("userId", userID)

	return service.Query(criteria, option.SortAsc("name"), option.SortAsc("price"))
}

// QueryByIDs returns a slice of Products that match the provided productIDs
func (service *Product) QueryByIDs(userID primitive.ObjectID, productIDs ...primitive.ObjectID) (sliceof.Object[model.Product], error) {

	criteria := exp.Equal("userId", userID).
		AndIn("_id", productIDs)

	return service.Query(criteria, option.SortAsc("name"), option.SortAsc("price"))
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Product) SyncRemoteProducts(userID primitive.ObjectID) (sliceof.Object[model.MerchantAccount], sliceof.Object[model.Product], error) {

	const location = "service.Product.SyncRemoteProducts"

	// Scan all Remote Products from every Merchant Account for this User
	merchantAccounts, remoteProducts, err := service.merchantAccountService.RemoteProductsByUser(userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error retrieving remote products for user", userID)
	}

	// If there are no Merchant Accounts, then there are no Remote Products
	if merchantAccounts.IsEmpty() {
		return merchantAccounts, sliceof.NewObject[model.Product](), nil
	}

	// Retrieve all Products currently in the database
	products, err := service.QueryByUser(userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error retrieving local products for user", userID)
	}

	productIndex := service.indexByRemoteID(products)

	for _, remoteProduct := range remoteProducts {

		// Skip this remote product if it already exists in the database
		if _, exists := productIndex[remoteProduct.RemoteID]; exists {
			continue
		}

		// Add the remote product to the database
		if err := service.Save(&remoteProduct, "Sync Remote Product"); err != nil {
			return nil, nil, derp.Wrap(err, location, "Error saving remote product", remoteProduct)
		}

		// Add the new remote product to the result
		products.Append(remoteProduct)
	}

	return merchantAccounts, products, nil
}

func (service *Product) indexByRemoteID(remoteProducts sliceof.Object[model.Product]) map[string]model.Product {

	// Create a map of remote IDs to Products
	result := make(map[string]model.Product, len(remoteProducts))

	for _, product := range remoteProducts {
		if product.RemoteID != "" {
			result[product.RemoteID] = product
		}
	}

	return result
}
