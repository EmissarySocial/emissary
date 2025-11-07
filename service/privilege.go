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
	circleService          *Circle
	identityService        *Identity
	merchantAccountService *MerchantAccount
}

// NewPrivilege returns a fully initialized Privilege service
func NewPrivilege() Privilege {
	return Privilege{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Privilege) Refresh(circleService *Circle, identityService *Identity, merchantAccountService *MerchantAccount) {
	service.circleService = circleService
	service.identityService = identityService
	service.merchantAccountService = merchantAccountService
}

// Close stops any background processes controlled by this service
func (service *Privilege) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Privilege) collection(session data.Session) data.Collection {
	return session.Collection("Privilege")
}

func (service *Privilege) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns an slice of allthe Privileges that match the provided criteria
func (service *Privilege) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Privilege, error) {
	result := make([]model.Privilege, 0)
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Privileges that match the provided criteria
func (service *Privilege) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Privilege records that match the provided criteria
func (service *Privilege) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Privilege], error) {

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Privilege.Range", "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewPrivilege), nil
}

// Load retrieves an Privilege from the database
func (service *Privilege) Load(session data.Session, criteria exp.Expression, privilege *model.Privilege) error {

	if err := service.collection(session).Load(notDeleted(criteria), privilege); err != nil {
		return derp.Wrap(err, "service.Privilege.Load", "Unable to load Privilege", criteria)
	}

	return nil
}

// Save adds/updates an Privilege in the database
func (service *Privilege) Save(session data.Session, privilege *model.Privilege, note string) error {

	const location = "service.Privilege.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(privilege); err != nil {
		return derp.Wrap(err, location, "Error validating Privilege", privilege)
	}

	// If the Identity does not exists, then creat a new Identity for this Privilege
	if err := service.maybeCreateIdentity(session, privilege); err != nil {
		return derp.Wrap(err, location, "Unable to create related Identity")
	}

	// RULE: Validate the CircleID for this Privilege
	if err := service.validateCircle(session, privilege); err != nil {
		return derp.Wrap(err, location, "Error validating Circle for Privilege", privilege)
	}

	// Save the privilege to the database
	if err := service.collection(session).Save(privilege, note); err != nil {
		return derp.Wrap(err, location, "Unable to save Privilege", privilege, note)
	}

	// Recalculate the privileges for the identityID
	if err := service.identityService.RefreshPrivileges(session, privilege.IdentityID); err != nil {

		if derp.IsNotFound(err) {
			privilege.IdentityID = primitive.NilObjectID
		} else {
			return derp.Wrap(err, location, "Error refreshing privileges", privilege.IdentityID)
		}
	}

	// Recalculate member counts for the Circle, if applicable
	if !privilege.CircleID.IsZero() {
		if err := service.circleService.RefreshMemberCounts(session, privilege.UserID, privilege.CircleID); err != nil {
			return derp.Wrap(err, location, "Error refreshing Circle member counts", privilege.CircleID)
		}
	}

	return nil
}

// Delete removes an Privilege from the database (virtual delete)
func (service *Privilege) Delete(session data.Session, privilege *model.Privilege, note string) error {

	const location = "service.Privilege.Delete"

	// Delete this Privilege
	if err := service.collection(session).Delete(privilege, note); err != nil {
		return derp.Wrap(err, location, "Error deleting Privilege", privilege, note)
	}

	// Recalculate the privileges for the identityID
	if err := service.identityService.RefreshPrivileges(session, privilege.IdentityID); err != nil {
		return derp.Wrap(err, location, "Error refreshing privileges", privilege.IdentityID)
	}

	// Recalculate member counts for the Circle, if applicable
	if !privilege.CircleID.IsZero() {
		if err := service.circleService.RefreshMemberCounts(session, privilege.UserID, privilege.CircleID); err != nil {
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

func (service *Privilege) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Privilege) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewPrivilege()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Privilege) ObjectSave(session data.Session, object data.Object, comment string) error {
	if privilege, ok := object.(*model.Privilege); ok {
		return service.Save(session, privilege, comment)
	}
	return derp.InternalError("service.Privilege.ObjectSave", "Invalid Object Type", object)
}

func (service *Privilege) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if privilege, ok := object.(*model.Privilege); ok {
		return service.Delete(session, privilege, comment)
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

func (service *Privilege) LoadByID(session data.Session, userID primitive.ObjectID, privilegeID primitive.ObjectID, privilege *model.Privilege) error {
	criteria := exp.Equal("_id", privilegeID).AndEqual("userId", userID)
	return service.Load(session, criteria, privilege)
}

func (service *Privilege) LoadByIdentity(session data.Session, identityID primitive.ObjectID, privilegeID primitive.ObjectID, privilege *model.Privilege) error {

	const location = "service.Privilege.LoadByIdentity"

	// RULE: IdentityID must not be zero
	if identityID.IsZero() {
		return derp.InternalError(location, "IdentityID must be provided")
	}

	// RULE: PrivilegeID must not be zero
	if privilegeID.IsZero() {
		return derp.InternalError(location, "PrivilegeID must be provided")
	}

	criteria := exp.Equal("_id", privilegeID).AndEqual("identityId", identityID)

	return service.Load(session, criteria, privilege)
}

func (service *Privilege) LoadByIdentityAndCircle(session data.Session, userID primitive.ObjectID, identityID primitive.ObjectID, circleID primitive.ObjectID, privilege *model.Privilege) error {

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

	return service.Load(session, criteria, privilege)
}

// RangeByIdentity returns an iterator containing all of the Privileges that match the provided IdentityID
func (service *Privilege) RangeByIdentity(session data.Session, identityID primitive.ObjectID) (iter.Seq[model.Privilege], error) {

	const location = "service.Privilege.RangeByIdentity"

	// RULE: IdentityID must not be zero
	if identityID.IsZero() {
		return nil, derp.InternalError(location, "IdentityID must be provided")
	}

	criteria := exp.Equal("identityId", identityID)

	return service.Range(session, criteria)
}

// RangeByIdentifiers returns an iterator containing all of the Privileges that match the provided identifiers (email, webfinger, activitypub)
func (service *Privilege) RangeByIdentifiers(session data.Session, emailAddress string, webfingerUsername string, activityPubActor string) (iter.Seq[model.Privilege], error) {

	// Create a criteria to find the Identity by any of the identifiers
	criteria := exp.Or(
		exp.And(
			exp.Equal("identifierType", model.IdentifierTypeEmail),
			exp.Equal("identifierValue", emailAddress),
		),
		exp.And(
			exp.Equal("identifierType", model.IdentifierTypeWebfinger),
			exp.Equal("identifierValue", webfingerUsername),
		),
		exp.And(
			exp.Equal("identifierType", model.IdentifierTypeActivityPub),
			exp.Equal("identifierValue", activityPubActor),
		),
	)

	return service.Range(session, criteria)
}

// RangeByCircle returns an iterator containing all of the Privileges that match the provided CircleID
func (service *Privilege) RangeByCircle(session data.Session, circleID primitive.ObjectID, options ...option.Option) (iter.Seq[model.Privilege], error) {

	const location = "service.Privilege.RangeByCircle"

	// RULE: CircleID must be provided
	if circleID.IsZero() {
		return nil, derp.InternalError(location, "No circleID provided")
	}

	criteria := exp.Equal("circleId", circleID)

	return service.Range(session, criteria, options...)
}

func (service *Privilege) RangeByProducts(session data.Session, productIDs ...primitive.ObjectID) (iter.Seq[model.Privilege], error) {

	const location = "service.Privilege.RangeByProductIDs"

	// RULE: Must have at least one productIDs
	if len(productIDs) == 0 {
		return nil, derp.InternalError(location, "No productIDs provided")
	}

	criteria := exp.In("productId", productIDs)
	return service.Range(session, criteria)
}

func (service *Privilege) QueryByIdentity(session data.Session, identityID primitive.ObjectID, options ...option.Option) ([]model.Privilege, error) {

	const location = "service.Privilege.QueryByIdentity"

	// RULE: IdentityID must be provided
	if identityID.IsZero() {
		return nil, derp.InternalError(location, "No identityID provided")
	}

	criteria := exp.Equal("identityId", identityID)

	return service.Query(session, criteria, options...)
}

// CountByIdentityAndCircle returns the number of privileges are granted to a particular Circle
func (service *Privilege) CountByCircle(session data.Session, circleID primitive.ObjectID) (int64, error) {
	criteria := exp.Equal("circleId", circleID)
	return service.Count(session, criteria)
}

// LoadByRemoteIDs retrieves a privilege using the remote IDs for the user, product, and privilege
func (service *Privilege) LoadByRemotePurchaseID(session data.Session, remotePurchaseID string, privilege *model.Privilege) error {
	criteria := exp.Equal("remotePurchaseId", remotePurchaseID)

	return service.Load(session, criteria, privilege)
}

/******************************************
 * Custom Behaviors
 ******************************************/

func (service *Privilege) DeleteByCircle(session data.Session, circleID primitive.ObjectID, note string) error {

	const location = "service.Circle.DeleteByCircle"

	// Range all privileges for this circle
	privileges, err := service.RangeByCircle(session, circleID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load Privileges for Circle", circleID)
	}

	// Delete them (thank you RangeFuncs!)
	for privilege := range privileges {

		// If this Privilege was purchased, then don't delete the purchase
		if privilege.RemotePurchaseID != "" {
			privilege.CircleID = primitive.NilObjectID // Remove the CircleID so that it is not counted in the future
			if err := service.collection(session).Save(&privilege, note); err != nil {
				return derp.Wrap(err, location, "Error removing CircleID from Privilege", privilege.ID(), note)
			}
			continue
		}

		// Otherwise, it's OK to delete an empty Privilege directly (no additional business logic)
		if err := service.collection(session).Delete(&privilege, note); err != nil {
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
func (service *Privilege) maybeCreateIdentity(session data.Session, privilege *model.Privilege) error {

	const location = "service.Privilege.maybeCreateIdentity"

	// RULE: If this privilege is already bound to a valid Identity, then we're all good
	if !privilege.IdentityID.IsZero() {
		return nil
	}

	// RULE: IdentifierType MUST be present
	if privilege.IdentifierValue == "" {
		return derp.BadRequestError(location, "Privilege must have an IdentifierValue to create an Identity", privilege)
	}

	// Try to guess the IdentifierType if it is not already set
	if privilege.IdentifierType == "" {
		privilege.IdentifierType = service.identityService.GuessIdentifierType(privilege.IdentifierValue)
	}

	// Fall through means we need to load or create the upstream record before we continue
	identity, err := service.identityService.LoadOrCreate(session, "", privilege.IdentifierType, privilege.IdentifierValue)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load/creating Identifier", privilege.IdentifierType, privilege.IdentifierValue)
	}

	// Update the Privilege with the correct IdentityID
	privilege.IdentityID = identity.IdentityID

	// If we are starting with a Webbfinger username, then switch to the ActivityPub actor before we save.
	if privilege.IdentifierType == model.IdentifierTypeWebfinger {
		privilege.IdentifierType = model.IdentifierTypeActivityPub
		privilege.IdentifierValue = identity.ActivityPubActor
	}

	return nil
}

func (service *Privilege) validateCircle(session data.Session, privilege *model.Privilege) error {

	const location = "service.Privilege.validateCircle"

	// If this privilege already has a CircleID, then we're done.
	if !privilege.CircleID.IsZero() {

		if privilege.Name == "" {

			circle := model.NewCircle()
			if err := service.circleService.LoadByID(session, privilege.UserID, privilege.CircleID, &circle); err != nil {
				return derp.Wrap(err, location, "Unable to load Circle by ID", privilege.CircleID)
			}

			privilege.SetCircleInfo(&circle)
		}
		return nil
	}

	// RULE: If the Privilege does not link to a ProductID, then we're done.
	if privilege.ProductID.IsZero() {
		return nil
	}

	// Since we've purchased a ProductID, let's see if there's a Circle that matches it.
	// If so, then we'll apply the CircleID to the Privilege.
	circle := model.NewCircle()
	if err := service.circleService.LoadByProductID(session, privilege.UserID, privilege.ProductID, &circle); err != nil {

		// If no Circle is bound to the RemoteProductID, then there's nothing to do.
		if derp.IsNotFound(err) {
			return nil
		}
		return derp.Wrap(err, location, "Unable to load Circle by RemoteProductID", privilege.RemoteProductID)
	}

	// Apply the CircleID to the Privilege
	privilege.CircleID = circle.CircleID
	return nil
}

// refreshIdentity recalculates all Privileges linked to the provided Identity,
// adding/removing IdentityIDs based on matching identifiers.
func (service *Privilege) refreshIdentity(session data.Session, identity *model.Identity) error {

	const location = "service.Privilege.RefreshIdentity"

	// RULE: NPE check
	if identity == nil {
		return derp.InternalError(location, "Identity cannot be nil.  This should never happen.")
	}

	// RULE: IdentityID must not be zero
	if identity.IdentityID.IsZero() {
		return derp.BadRequestError(location, "IdentityID cannot be empty.  This should never happen.", identity)
	}

	//////////////////////////
	// Step 1: Remove the IdentityID from Privileges
	// that no longer match the identifiers for this Identity

	// Load all of the Privileges that match this IdentityID
	privilegesToRemove, err := service.RangeByIdentity(session, identity.IdentityID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load Privileges by IdentityID", identity.IdentityID)
	}

	// Remove this Identity from all privileges that no longer have matching identifiers
	for privilege := range privilegesToRemove {
		if err := service.maybeRemoveIdentity(session, &privilege, identity); err != nil {
			return derp.Wrap(err, location, "Unable to remove IdentityID from Privilege", privilege.PrivilegeID)
		}
	}

	//////////////////////////
	// Step 2: Reassign the IdentityID to all Privileges
	// that currently match the identifiers for this Identity

	// Load all Privileges that match the identifiers for this Identity
	privilegesToAssign, err := service.RangeByIdentifiers(session, identity.EmailAddress, identity.WebfingerUsername, identity.ActivityPubActor)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load Privileges by identifiers", identity)
	}

	for privilege := range privilegesToAssign {

		if err := service.maybeSetIdentity(session, &privilege, identity); err != nil {
			return derp.Wrap(err, location, "Unable to set IdentityID on Privilege", privilege.PrivilegeID)
		}
	}

	// Phew!  Everything is awesome.
	return nil
}

func (service *Privilege) RefreshCircleInfo(session data.Session, circle *model.Circle) error {

	const location = "service.Privilege.RefreshCircle"

	// Set CircleID for all Privileges that match the Products linked to this Circle
	if circle.ProductIDs.NotEmpty() {

		privileges, err := service.RangeByProducts(session, circle.ProductIDs...)

		if err != nil {
			return derp.Wrap(err, location, "Unable to load Privileges by RemoteTokens", circle.CircleID)
		}

		for privilege := range privileges {

			// Update the Circle if it does not match
			if privilege.SetCircleInfo(circle) {

				if err := service.Save(session, &privilege, "Updating Circle settings"); err != nil {
					return derp.Wrap(err, location, "Error refreshing Privilege", circle.CircleID)
				}
			}
		}
	}

	// Remove CircleID from all Privileges that no longer match the ProductIDs linked to this Circle
	privileges, err := service.RangeByCircle(session, circle.CircleID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to load Privileges by CircleID", circle)
	}

	for privilege := range privileges {

		// If the Privilege is linked to the Circle incorrectly, then remove the CircleID
		if privilege.IsPurchase() && !circle.ProductIDs.Contains(privilege.ProductID) {
			privilege.CircleID = primitive.NilObjectID

			if err := service.Save(session, &privilege, "Updating Circle settings"); err != nil {
				return derp.Wrap(err, location, "Error refreshing Privilege", circle)
			}

			continue
		}

		// If the Circle info has changed, then update the Privilege
		if privilege.SetCircleInfo(circle) {
			if err := service.Save(session, &privilege, "Updating Circle settings"); err != nil {
				return derp.Wrap(err, location, "Error refreshing Privilege", circle)
			}
		}
	}

	return nil
}

// maybeSetIdentity bi-directionally links a Privilege to an Identity
func (service *Privilege) maybeSetIdentity(session data.Session, privilege *model.Privilege, identity *model.Identity) error {

	const location = "service.Privilege.SetIdentityID"

	// RULE: Privilege must not be nil
	if privilege == nil {
		return derp.BadRequestError(location, "Privilege cannot be nil. This should never happen.")
	}

	// RULE: Identity must not be nil
	if identity == nil {
		return derp.BadRequestError(location, "Identity cannot be nil. This should never happen.")
	}

	// If the identifier does not match, then do not reassign (but this should never happen)
	if identity.Identifier(privilege.IdentifierType) != privilege.IdentifierValue {
		return derp.BadRequestError(location, "Privilege must match the identifier in the Identity. This shoulld never happen.", privilege.IdentifierType, privilege.IdentifierValue, identity)
	}

	// Make sure that the Identity includes a link to the Privilege
	identity.SetPrivilegeID(privilege.PrivilegeID)

	// If the Privilege is already linked to this Identity, then there's nothing more to do.
	if privilege.IdentityID == identity.IdentityID {
		return nil
	}

	// Set the IdentityID in the Privilege
	privilege.IdentityID = identity.IdentityID

	// Update the Privilege without triggering any additional business logic.
	if err := service.collection(session).Save(privilege, "Setting IdentityID"); err != nil {
		return derp.Wrap(err, location, "Unable to set IdentityID on Privilege", privilege.PrivilegeID)
	}

	// Return in success.
	return nil
}

func (service *Privilege) maybeRemoveIdentity(session data.Session, privilege *model.Privilege, identity *model.Identity) error {

	const location = "service.Privilege.RemoveIdentity"

	// RULE: Privilege must not be nil
	if privilege == nil {
		return derp.BadRequestError(location, "Privilege cannot be nil")
	}

	// RULE: Identity must not be nil
	if identity == nil {
		return derp.BadRequestError(location, "Identity cannot be nil. This should never happen")
	}

	// If the identifier matches, then this Privilege is still valid
	if identity.Identifier(privilege.IdentifierType) == privilege.IdentifierValue {
		return nil
	}

	// Remove the PrivilegeID from the Identity
	identity.RemovePrivilegeID(privilege.PrivilegeID)

	// RULE: If the Identity is already Zero, then there's nothing to do
	if privilege.IdentityID.IsZero() {
		return nil
	}

	// Remove the IdentityID from the Privilege
	privilege.IdentityID = primitive.NilObjectID

	if err := service.collection(session).Save(privilege, "Removing IdentityID"); err != nil {
		return derp.Wrap(err, location, "Unable to remove IdentityID from Privilege", privilege.PrivilegeID)
	}

	return nil
}

// Cancel cancels a Privilege subscription.
func (service *Privilege) Cancel(session data.Session, privilege *model.Privilege) error {

	const location = "service.Privilege.Cancel"

	if privilege.MerchantAccountID.IsZero() {
		return derp.BadRequestError(location, "Privilege cannot be canceled without a valid MerchantAccountID")
	}

	if !privilege.IsRecurring() {
		return derp.BadRequestError(location, "Privilege cannot be canceled if it is not a recurring charge.")
	}

	if err := service.merchantAccountService.CancelPrivilege(session, privilege); err != nil {
		return derp.Wrap(err, location, "Error canceling subscription for Privilege", privilege.PrivilegeID)
	}

	if err := service.Delete(session, privilege, "Canceled by User"); err != nil {
		return derp.Wrap(err, location, "Error deleting Privilege after canceling subscription", privilege.PrivilegeID)
	}

	return nil
}
