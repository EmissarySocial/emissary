package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/schema"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Privilege defines a service that manages all content privileges created and imported by Users.
type Privilege struct {
	collection      data.Collection
	identityService *Identity
}

// NewPrivilege returns a fully initialized Privilege service
func NewPrivilege() Privilege {
	return Privilege{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Privilege) Refresh(collection data.Collection, identityService *Identity) {
	service.collection = collection
	service.identityService = identityService
}

// Close stops any background processes controlled by this service
func (service *Privilege) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Privilege) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Privileges that match the provided criteria
func (service *Privilege) Query(criteria exp.Expression, options ...option.Option) ([]model.Privilege, error) {
	result := make([]model.Privilege, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Privileges that match the provided criteria
func (service *Privilege) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Privilege records that match the provided criteria
func (service *Privilege) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Privilege], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Privilege.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewPrivilege), nil
}

// Load retrieves an Privilege from the database
func (service *Privilege) Load(criteria exp.Expression, privilege *model.Privilege) error {

	if err := service.collection.Load(notDeleted(criteria), privilege); err != nil {
		return derp.Wrap(err, "service.Privilege.Load", "Error loading Privilege", criteria)
	}

	return nil
}

// Save adds/updates an Privilege in the database
func (service *Privilege) Save(privilege *model.Privilege, note string) error {

	const location = "service.Privilege.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(privilege); err != nil {
		return derp.Wrap(err, location, "Error validating Privilege", privilege)
	}

	// Save the privilege to the database
	if err := service.collection.Save(privilege, note); err != nil {
		return derp.Wrap(err, location, "Error saving Privilege", privilege, note)
	}

	// Recalculate the privileges for the identityID
	if err := service.identityService.RefreshPrivileges(privilege.IdentityID); err != nil {
		return derp.Wrap(err, location, "Error refreshing privileges", privilege.IdentityID)
	}

	return nil
}

// Delete removes an Privilege from the database (virtual delete)
func (service *Privilege) Delete(privilege *model.Privilege, note string) error {

	const location = "service.Privilege.Delete"

	// Delete this Privilege
	if err := service.collection.Delete(privilege, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Privilege", privilege, note)
	}

	// Recalculate the privileges for the identityID
	if err := service.identityService.RefreshPrivileges(privilege.IdentityID); err != nil {
		return derp.Wrap(err, location, "Error refreshing privileges", privilege.IdentityID)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Privilege) ObjectType() string {
	return "Privilege"
}

// New returns a fully initialized model.Privilege as a data.Object.
func (service *Privilege) ObjectNew() data.Object {
	result := model.NewPrivilege()
	return &result
}

func (service *Privilege) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Privilege); ok {
		return mention.PrivilegeID
	}

	return primitive.NilObjectID
}

func (service *Privilege) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Privilege) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewPrivilege()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Privilege) ObjectSave(object data.Object, comment string) error {
	if privilege, ok := object.(*model.Privilege); ok {
		return service.Save(privilege, comment)
	}
	return derp.InternalError("service.Privilege.ObjectSave", "Invalid Object Type", object)
}

func (service *Privilege) ObjectDelete(object data.Object, comment string) error {
	if privilege, ok := object.(*model.Privilege); ok {
		return service.Delete(privilege, comment)
	}
	return derp.InternalError("service.Privilege.ObjectDelete", "Invalid Object Type", object)
}

func (service *Privilege) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Privilege.ObjectUserCan", "Not Authorized")
}

func (service *Privilege) Schema() schema.Schema {
	return schema.New(model.PrivilegeSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Privilege) QueryByIdentity(identityID primitive.ObjectID, options ...option.Option) ([]model.Privilege, error) {

	const location = "service.Privilege.QueryByIdentity"

	// RULE: IdentityID must be provided
	if identityID.IsZero() {
		return nil, derp.InternalError(location, "No identityID provided")
	}

	criteria := exp.Equal("identityId", identityID)

	return service.Query(criteria, options...)
}

// CountByIdentityAndProduct returns the number of privileges made by an identity for a list of products
func (service *Privilege) CountByIdentityAndProduct(identityID primitive.ObjectID, remoteProductIDs ...string) (int64, error) {

	const location = "service.Privilege.CountByIdentityAndProduct"

	// RULE: IdentityID must be provided
	if identityID.IsZero() {
		return 0, derp.InternalError(location, "No identityID provided")
	}

	// RULE: At least one productID must be provided
	if len(remoteProductIDs) == 0 {
		return 0, derp.InternalError(location, "No productIDs provided")
	}

	criteria := exp.Equal("identityId", identityID).AndIn("remoteProductId", remoteProductIDs)

	return service.Count(criteria)
}

// LoadByRemoteIDs retrieves a privilege using the remote IDs for the user, product, and privilege
func (service *Privilege) LoadByRemoteIDs(remotePersonID string, remoteProductID string, privilege *model.Privilege) error {
	criteria := exp.Equal("remotePersonId", remotePersonID).
		AndEqual("remoteProductId", remoteProductID)

	return service.Load(criteria, privilege)
}
