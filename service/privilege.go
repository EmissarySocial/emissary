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
	circleService   *Circle
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
func (service *Privilege) Refresh(collection data.Collection, circleService *Circle, identityService *Identity) {
	service.collection = collection
	service.circleService = circleService
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

	// If the Identity does not exists, then creat a new Identity for this Privilege
	if err := service.maybeCreateIdentity(privilege); err != nil {
		return derp.Wrap(err, location, "Error creating related Identity")
	}

	// RULE: Validate the CircleID for this Privilege
	if err := service.validateCircle(privilege); err != nil {
		return derp.Wrap(err, location, "Error validating Circle for Privilege", privilege)
	}

	// Save the privilege to the database
	if err := service.collection.Save(privilege, note); err != nil {
		return derp.Wrap(err, location, "Error saving Privilege", privilege, note)
	}

	// Recalculate the privileges for the identityID
	if err := service.identityService.RefreshPrivileges(privilege.IdentityID); err != nil {
		return derp.Wrap(err, location, "Error refreshing privileges", privilege.IdentityID)
	}

	// Recalculate member counts for the Circle, if applicable
	if !privilege.CircleID.IsZero() {
		if err := service.circleService.RefreshMemberCounts(privilege.UserID, privilege.CircleID); err != nil {
			return derp.Wrap(err, location, "Error refreshing Circle member counts", privilege.CircleID)
		}
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

	// Recalculate member counts for the Circle, if applicable
	if !privilege.CircleID.IsZero() {
		if err := service.circleService.RefreshMemberCounts(privilege.UserID, privilege.CircleID); err != nil {
			return derp.Wrap(err, location, "Error refreshing Circle member counts", privilege.CircleID)
		}
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

func (service *Privilege) LoadByID(userID primitive.ObjectID, privilegeID primitive.ObjectID, privilege *model.Privilege) error {
	criteria := exp.Equal("_id", privilegeID).AndEqual("userId", userID)
	return service.Load(criteria, privilege)
}

func (service *Privilege) LoadByIdentityAndCircle(userID primitive.ObjectID, identityID primitive.ObjectID, circleID primitive.ObjectID, privilege *model.Privilege) error {

	const location = "service.Privilege.LoadByIdentityAndCircle"

	// RULE: UserID must not be zero
	if userID.IsZero() {
		return derp.InternalError(location, "UserID must be provided")
	}

	// RULE: CircleID must not be zero
	if identityID.IsZero() {
		return derp.InternalError(location, "IdentityID must be provided")
	}

	// RULE: CircleID must not be zero
	if circleID.IsZero() {
		return derp.InternalError(location, "CircleID must be provided")
	}

	criteria := exp.Equal("userId", userID).
		AndEqual("identityId", identityID).
		AndEqual("circleId", circleID)

	return service.Load(criteria, privilege)
}

func (service *Privilege) RangeByCircle(circleID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Privilege], error) {

	const location = "service.Privilege.RangeByCircle"

	// RULE: CircleID must be provided
	if circleID.IsZero() {
		return nil, derp.InternalError(location, "No circleID provided")
	}

	criteria := exp.Equal("circleId", circleID)

	return service.Range(criteria, options...)
}

func (service *Privilege) RangeByProducts(productIDs ...primitive.ObjectID) (iter.Seq[model.Privilege], error) {

	const location = "service.Privilege.RangeByProductIDs"

	// RULE: Must have at least one productIDs
	if len(productIDs) == 0 {
		return nil, derp.InternalError(location, "No productIDs provided")
	}

	criteria := exp.In("productId", productIDs)
	return service.Range(criteria)
}

func (service *Privilege) QueryByIdentity(identityID primitive.ObjectID, options ...option.Option) ([]model.Privilege, error) {

	const location = "service.Privilege.QueryByIdentity"

	// RULE: IdentityID must be provided
	if identityID.IsZero() {
		return nil, derp.InternalError(location, "No identityID provided")
	}

	criteria := exp.Equal("identityId", identityID)

	return service.Query(criteria, options...)
}

// CountByIdentityAndCircle returns the number of privileges are granted to a particular Circle
func (service *Privilege) CountByCircle(circleID primitive.ObjectID) (int64, error) {
	criteria := exp.Equal("circleId", circleID)
	return service.Count(criteria)
}

// LoadByRemoteIDs retrieves a privilege using the remote IDs for the user, product, and privilege
func (service *Privilege) LoadByRemotePurchaseID(remotePurchaseID string, privilege *model.Privilege) error {
	criteria := exp.Equal("remotePurchaseId", remotePurchaseID)

	return service.Load(criteria, privilege)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Privilege) DeleteByCircle(circleID primitive.ObjectID, note string) error {

	const location = "service.Circle.DeleteByCircle"

	// Range all privileges for this circle
	privileges, err := service.RangeByCircle(circleID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading Privileges for Circle", circleID)
	}

	// Delete them (thank you RangeFuncs!)
	for privilege := range privileges {
		if err := service.Delete(&privilege, note); err != nil {
			return derp.Wrap(err, location, "Error deleting Privilege", privilege.ID(), note)
		}
	}

	// Everything is awesome.
	return nil
}

/******************************************
 * Helper Methods
 ******************************************/

// maybeCreateIdentity guarantees that the provided Privilege is connected to a valid Identity.
// If the IdentityID field is zero, then a matching Identity is located or created.
func (service *Privilege) maybeCreateIdentity(privilege *model.Privilege) error {

	const location = "service.Privilege.maybeCreateIdentity"

	// If this privilege is already bound to a valid Identity, then we're all good
	if !privilege.IdentityID.IsZero() {
		return nil
	}

	// Fall through means we need to load or create the upstream record before we continue
	identity, err := service.identityService.LoadOrCreate("", privilege.IdentifierType, privilege.IdentifierValue)

	if err != nil {
		return derp.Wrap(err, location, "Error loading/creating Identifier", privilege.IdentifierType, privilege.IdentifierValue)
	}

	// Update the Privilege with the correct IdentityID
	privilege.IdentityID = identity.IdentityID

	return nil
}

func (service *Privilege) validateCircle(privilege *model.Privilege) error {

	// If this privilege already has a CircleID, then we're done.
	if !privilege.CircleID.IsZero() {
		return nil
	}

	// RULE: If the Privilege does not link to a ProductID, then we're done.
	if privilege.ProductID.IsZero() {
		return nil
	}

	// Since we've purchased a ProductID, let's see if there's a Circle that matches it.
	// If so, then we'll apply the CircleID to the Privilege.
	circle := model.NewCircle()
	if err := service.circleService.LoadByProductID(privilege.UserID, privilege.ProductID, &circle); err != nil {

		// If no Circle is bound to the RemoteProductID, then there's nothing to do.
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, "service.Privilege.validateCircle", "Error loading Circle by RemoteProductID", privilege.RemoteProductID)
	}

	// Apply the CircleID to the Privilege
	privilege.CircleID = circle.CircleID
	return nil
}

func (service *Privilege) RefreshCircle(circle *model.Circle) error {

	const location = "service.Privilege.RefreshCircle"

	// Set CircleID for all Privileges that match the Products linked to this Circle
	if circle.ProductIDs.NotEmpty() {

		privileges, err := service.RangeByProducts(circle.ProductIDs...)

		if err != nil {
			return derp.Wrap(err, location, "Error loading Privileges by RemoteTokens", circle.CircleID)
		}

		for privilege := range privileges {

			// Update the Circle if it does not match
			if privilege.CircleID != circle.CircleID {
				privilege.CircleID = circle.CircleID
				if err := service.Save(&privilege, "Updating Circle settings"); err != nil {
					return derp.Wrap(err, location, "Error refreshing Privilege", circle.CircleID)
				}
			}
		}
	}

	// Remove CircleID from all Privileges that no longer match the ProductIDs linked to this Circle
	privileges, err := service.RangeByCircle(circle.CircleID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading Privileges by CircleID", circle)
	}

	for privilege := range privileges {

		if circle.ProductIDs.NotContains(privilege.ProductID) {
			privilege.CircleID = primitive.NilObjectID
			if err := service.Save(&privilege, "Updating Circle settings"); err != nil {
				return derp.Wrap(err, location, "Error refreshing Privilege", circle)
			}
		}
	}

	return nil
}
