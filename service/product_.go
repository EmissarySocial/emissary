package service

import (
	"iter"

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
	importItemService      *ImportItem
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
func (service *Product) Refresh(importItemService *ImportItem, merchantAccountService *MerchantAccount) {
	service.importItemService = importItemService
	service.merchantAccountService = merchantAccountService
}

// Close stops any background processes controlled by this service
func (service *Product) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Product) collection(session data.Session) data.Collection {
	return session.Collection("Product")
}

// Count returns the number of records that match the provided criteria
func (service *Product) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of Products that match the provided criteria
func (service *Product) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Product, error) {
	result := make([]model.Product, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Products who match the provided criteria
func (service *Product) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Products that match the provided criteria
func (service *Product) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Product], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Product.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewProduct), nil
}

// Load retrieves an Product from the database
func (service *Product) Load(session data.Session, criteria exp.Expression, result *model.Product) error {
	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Product.Load", "Unable to load Product", criteria)
	}

	return nil
}

// Save adds/updates an Product in the database
func (service *Product) Save(session data.Session, product *model.Product, note string) error {

	const location = "service.Product.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(product); err != nil {
		return derp.Wrap(err, location, "Invalid Product", product)
	}

	// Save the value to the database
	if err := service.collection(session).Save(product, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Product", product, note)
	}

	return nil
}

// Delete removes an Product from the database (virtual delete)
func (service *Product) Delete(session data.Session, product *model.Product, note string) error {

	if err := service.collection(session).Delete(product, note); err != nil {
		return derp.Wrap(err, "service.Product.Delete", "Unable to delete Product", product, note)
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly documents that match the provided criteria
func (service *Product) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Product record, without applying any additional business rules
func (service *Product) HardDeleteByID(session data.Session, userID primitive.ObjectID, productID primitive.ObjectID) error {

	const location = "service.Product.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", productID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Unable to delete Product", "userID: "+userID.Hex(), "productID: "+productID.Hex())
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

func (service *Product) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Product) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewProduct()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Product) ObjectSave(session data.Session, object data.Object, comment string) error {
	if product, ok := object.(*model.Product); ok {
		return service.Save(session, product, comment)
	}
	return derp.InternalError("service.Product.ObjectSave", "Invalid Object Type", object)
}

func (service *Product) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if product, ok := object.(*model.Product); ok {
		return service.Delete(session, product, comment)
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
func (service *Product) LoadByID(session data.Session, productID primitive.ObjectID, result *model.Product) error {
	criteria := exp.Equal("_id", productID)
	return service.Load(session, criteria, result)
}

// LoadByUserAndID loads a single model.Product object that matches the provided userID and productID
func (service *Product) LoadByUserAndID(session data.Session, userID primitive.ObjectID, productID primitive.ObjectID, result *model.Product) error {
	criteria := exp.Equal("userID", userID).AndEqual("_id", productID)
	return service.Load(session, criteria, result)
}

func (service *Product) LoadByToken(session data.Session, token string, result *model.Product) error {

	productID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Product.LoadByToken", "Invalid Product ID", token)
	}

	return service.LoadByID(session, productID, result)
}

// LoadByRemoteID loads a single model.Product object that matches the provided remoteID
func (service *Product) LoadByRemoteID(session data.Session, userID primitive.ObjectID, remoteID string, result *model.Product) error {

	criteria := exp.Equal("userId", userID).AndEqual("remoteId", remoteID)
	return service.Load(session, criteria, result)
}

// QueryByUser returns a slice of Products that match the provided productIDs
func (service *Product) QueryByUser(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.Product], error) {

	criteria := exp.Equal("userId", userID)

	return service.Query(session, criteria, option.SortAsc("name"), option.SortAsc("price"))
}

// QueryByIDs returns a slice of Products that match the provided productIDs
func (service *Product) QueryByIDs(session data.Session, userID primitive.ObjectID, productIDs ...primitive.ObjectID) (sliceof.Object[model.Product], error) {

	criteria := exp.Equal("userId", userID).
		AndIn("_id", productIDs)

	return service.Query(session, criteria, option.SortAsc("name"), option.SortAsc("price"))
}

// RangeByUserID returns a RangeFunc that yields all Products owned by the provided UserID
func (service *Product) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Product], error) {
	criteria := exp.Equal("userId", userID)
	return service.Range(session, criteria)
}

// DeleteByUserID deletes all Products owned by the provided UserID
func (service *Product) DeleteByUserID(session data.Session, userID primitive.ObjectID, note string) error {

	const location = "service.Product.DeleteByUserID"

	// Retrieve all Products
	products, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to query products by UserID", userID)
	}

	// Delete each product
	for product := range products {
		if err := service.Delete(session, &product, note); err != nil {
			return derp.Wrap(err, location, "Unable to delete Product", product)
		}
	}

	// Success
	return nil
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Product) SyncRemoteProducts(session data.Session, userID primitive.ObjectID) (sliceof.Object[model.MerchantAccount], sliceof.Object[model.Product], error) {

	const location = "service.Product.SyncRemoteProducts"

	// Scan all Remote Products from every Merchant Account for this User
	merchantAccounts, remoteProducts, err := service.merchantAccountService.RemoteProductsByUser(session, userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error retrieving remote products for user", userID)
	}

	// If there are no Merchant Accounts, then there are no Remote Products
	if merchantAccounts.IsEmpty() {
		return merchantAccounts, sliceof.NewObject[model.Product](), nil
	}

	// Retrieve all Products currently in the database
	products, err := service.QueryByUser(session, userID)

	if err != nil {
		return nil, nil, derp.Wrap(err, location, "Error retrieving local products for user", userID)
	}

	productIndex := service.indexByRemoteID(products)

	result := make(sliceof.Object[model.Product], 0, len(remoteProducts))

	// Scan products/remoteProducts; add new remote Products to local Products list.
	for _, remoteProduct := range remoteProducts {

		// Skip this remote product if it already exists in the database
		if currentProduct, exists := productIndex[remoteProduct.RemoteID]; exists {

			// Remote the current product from the Index so we won't delete it later
			delete(productIndex, remoteProduct.RemoteID)

			// If the remote product changed, then update the local product record
			changed, err := currentProduct.Refresh(remoteProduct)

			if err != nil {
				return nil, nil, derp.Wrap(err, location, "Unable to refresh remote product", remoteProduct)
			}

			if changed {
				if err := service.Save(session, &currentProduct, "Updated Remote Product changes"); err != nil {
					return nil, nil, derp.Wrap(err, location, "Unable to save updated remote product", currentProduct)
				}
			}

			// Add the existing product to the result
			result.Append(currentProduct)
			continue
		}

		// Add the remote product to the database
		if err := service.Save(session, &remoteProduct, "Sync Remote Product"); err != nil {
			return nil, nil, derp.Wrap(err, location, "Unable to save remote product", remoteProduct)
		}

		// Add the new Remote Product to the result
		result.Append(remoteProduct)
	}

	// Remove local Product records that are no longer in the remote products list
	for _, product := range productIndex {
		if err := service.Delete(session, &product, "Removed from merchant account"); err != nil {
			return nil, nil, derp.Wrap(err, location, "Unable to delete local product", product)
		}
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
