package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/schema"
	"github.com/benpate/rosetta/slice"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Identity defines a service that manages all content identitys created and imported by Users.
type Identity struct {
	collection       data.Collection
	privilegeService *Privilege
}

// NewIdentity returns a fully initialized Identity service
func NewIdentity() Identity {
	return Identity{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Identity) Refresh(collection data.Collection, privilegeService *Privilege) {
	service.collection = collection
	service.privilegeService = privilegeService
}

// Close stops any background processes controlled by this service
func (service *Identity) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Identity) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Identitys that match the provided criteria
func (service *Identity) Query(criteria exp.Expression, options ...option.Option) ([]model.Identity, error) {
	result := make([]model.Identity, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Identitys that match the provided criteria
func (service *Identity) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Identity records that match the provided criteria
func (service *Identity) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Identity], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Identity.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewIdentity), nil
}

// Load retrieves an Identity from the database
func (service *Identity) Load(criteria exp.Expression, identity *model.Identity) error {

	if err := service.collection.Load(notDeleted(criteria), identity); err != nil {
		return derp.Wrap(err, "service.Identity.Load", "Error loading Identity", criteria)
	}

	return nil
}

// Save adds/updates an Identity in the database
func (service *Identity) Save(identity *model.Identity, note string) error {

	// Pick a default name, if necessary
	if identity.Name == "" {
		identity.Name = first.String(identity.EmailAddress, identity.WebFingerHandle)
	}

	// Validate the value before saving
	if err := service.Schema().Validate(identity); err != nil {
		return derp.Wrap(err, "service.Identity.Save", "Error validating Identity", identity)
	}

	// Save the identity to the database
	if err := service.collection.Save(identity, note); err != nil {
		return derp.Wrap(err, "service.Identity.Save", "Error saving Identity", identity, note)
	}

	return nil
}

// Delete removes an Identity from the database (virtual delete)
func (service *Identity) Delete(identity *model.Identity, note string) error {

	// Delete this Identity
	if err := service.collection.Delete(identity, note); err != nil {
		return derp.Wrap(err, "service.Identity.Delete", "Error deleting Identity", identity, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Identity) ObjectType() string {
	return "Identity"
}

// New returns a fully initialized model.Identity as a data.Object.
func (service *Identity) ObjectNew() data.Object {
	result := model.NewIdentity()
	return &result
}

func (service *Identity) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Identity); ok {
		return mention.IdentityID
	}

	return primitive.NilObjectID
}

func (service *Identity) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Identity) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewIdentity()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Identity) ObjectSave(object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Save(identity, comment)
	}
	return derp.InternalError("service.Identity.ObjectSave", "Invalid Object Type", object)
}

func (service *Identity) ObjectDelete(object data.Object, comment string) error {
	if identity, ok := object.(*model.Identity); ok {
		return service.Delete(identity, comment)
	}
	return derp.InternalError("service.Identity.ObjectDelete", "Invalid Object Type", object)
}

func (service *Identity) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Identity.ObjectUserCan", "Not Authorized")
}

func (service *Identity) Schema() schema.Schema {
	return schema.New(model.IdentitySchema())
}

/******************************************
 * Custom Queries
 ******************************************/

func (service *Identity) LoadByID(identityID primitive.ObjectID, identity *model.Identity) error {

	if identityID.IsZero() {
		return derp.BadRequestError("service.Identity.LoadByID", "IdentityID cannot be empty", identityID)
	}

	criteria := exp.Equal("_id", identityID)
	return service.Load(criteria, identity)
}

// LoadOrCreateByEmail searches for a Guest with the provided emailAddress.
// If a matching record is found, it updates the record with the new values (if necessary).
// If no matching record is found, it creates a new record with the provided values.
func (service *Identity) LoadOrCreate(identifierType string, identifierValue string, isVerified bool) (model.Identity, error) {

	const location = "service.Identity.LoadOrCreate"

	// RULE: Identifier Type must be provided
	if identifierType == "" {
		return model.Identity{}, derp.InternalError(location, "Identifier type cannot be empty")
	}

	// RULE: Identifier Value must be provided
	if identifierValue == "" {
		return model.Identity{}, derp.InternalError(location, "Identifier value cannot be empty")
	}

	// Try to load the Identity using the provided identifier
	identity := model.NewIdentity()
	err := service.LoadByIdentifier(identifierType, identifierValue, &identity)

	// If the identity was found, then just return it...
	if err == nil {

		// ... but before we do, there's a slight chance we may need to "verify" the identifier first

		// If the caller has verified the identifier but it's not verified in the Identity, then verify it now
		if isVerified {

			if changed := identity.Verify(identifierType); changed {

				if err := service.Save(&identity, "Verified Identifier"); err != nil {
					return model.Identity{}, derp.Wrap(err, location, "Error verifying identity", identity)
				}
			}
		}

		return identity, nil
	}

	// If the error was anything but "not found", then return the error
	if !derp.IsNotFound(err) {
		return model.Identity{}, derp.Wrap(err, location, "Error loading identity", identifierType, identifierValue)
	}

	// Otherwise, populate the identifier into the Identity object
	if ok := identity.SetIdentifier(identifierType, identifierValue); !ok {
		return model.Identity{}, derp.BadRequestError(location, "Invalid Identifier Type", identifierType)
	}

	// If the identifier has been verified
	if isVerified {
		identity.Verify(identifierType)
	}

	// Save the Identity to the database
	if err := service.Save(&identity, "Updated"); err != nil {
		return model.Identity{}, derp.Wrap(err, location, "Error saving identity", identity)
	}

	// Done.
	return identity, nil
}

func (service *Identity) LoadByIdentifier(identifierType string, identifierValue string, identity *model.Identity) error {

	switch identifierType {

	case model.IdentifierTypeEmail:
		return service.LoadByEmailAddress(identifierValue, identity)

	case model.IdentifierTypeWebFinger:
		return service.LoadByWebFingerHandle(identifierValue, identity)
	}

	return derp.InternalError("service.Identity.LoadByAddress", "Invalid Identity Type", identifierType)
}

// LoadByEmail retrieves a single Identity from the database using the provided email address
func (service *Identity) LoadByEmailAddress(emailAddress string, identity *model.Identity) error {
	criteria := exp.Equal("emailAddress", emailAddress)
	return service.Load(criteria, identity)
}

// LoadByWebFingerHandle retrieves a single Identity from the database using the provided WebFinger handle
func (service *Identity) LoadByWebFingerHandle(emailAddress string, identity *model.Identity) error {
	criteria := exp.Equal("webfingerHandle", emailAddress)
	return service.Load(criteria, identity)
}

// RefreshPrivileges recalculates the privileges for the provided IdentityID by loading all privileges
// and collecting the list of unique CircleIDs and RemoteProductIDs.
func (service *Identity) RefreshPrivileges(identityID primitive.ObjectID) error {

	const location = "service.Identity.RefreshPrivileges"

	// Load the Identity from the database
	identity := model.NewIdentity()
	if err := service.LoadByID(identityID, &identity); err != nil {
		return derp.Wrap(err, location, "Error loading identity", identityID)
	}

	// Get all privileges for this Identity
	privileges, err := service.privilegeService.QueryByIdentity(identityID)

	if err != nil {
		return derp.Wrap(err, location, "Error loading privileges for identity", identityID)
	}

	// Collect the CircleIDs and RemoteProductIDs from each privileges
	privilegeStrings := make([]string, 0, len(privileges))

	for _, privilege := range privileges {

		if !privilege.CircleID.IsZero() {
			privilegeStrings = append(privilegeStrings, privilege.CircleID.Hex())
		}

		if privilege.RemoteProductID != "" {
			privilegeStrings = append(privilegeStrings, privilege.RemoteProductID)
		}
	}

	// Remove Duplicates
	privilegeStrings = slice.Unique(privilegeStrings)

	// Apply to Identity
	identity.Privileges = privilegeStrings

	// Save changes
	if err := service.Save(&identity, "Refreshed Privileges"); err != nil {
		return derp.Wrap(err, location, "Error saving identity", identity)
	}

	// Retire in Cabo
	return nil
}
