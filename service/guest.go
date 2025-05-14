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

// Guest defines a service that manages all content guests created and imported by Users.
type Guest struct {
	collection data.Collection
}

// NewGuest returns a fully initialized Guest service
func NewGuest() Guest {
	return Guest{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Guest) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Guest) Close() {
	// Nothin to do here.
}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Guest) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns an slice of allthe Guests that match the provided criteria
func (service *Guest) Query(criteria exp.Expression, options ...option.Option) ([]model.Guest, error) {
	result := make([]model.Guest, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)

	return result, err
}

// List returns an iterator containing all of the Guests that match the provided criteria
func (service *Guest) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Range returns a Go 1.23 RangeFunc that iterates over the Guest records that match the provided criteria
func (service *Guest) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Guest], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Guest.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewGuest), nil
}

// Load retrieves an Guest from the database
func (service *Guest) Load(criteria exp.Expression, guest *model.Guest) error {

	if err := service.collection.Load(notDeleted(criteria), guest); err != nil {
		return derp.Wrap(err, "service.Guest.Load", "Error loading Guest", criteria)
	}

	return nil
}

// Save adds/updates an Guest in the database
func (service *Guest) Save(guest *model.Guest, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(guest); err != nil {
		return derp.Wrap(err, "service.Guest.Save", "Error validating Guest", guest)
	}

	// Save the guest to the database
	if err := service.collection.Save(guest, note); err != nil {
		return derp.Wrap(err, "service.Guest.Save", "Error saving Guest", guest, note)
	}

	return nil
}

// Delete removes an Guest from the database (virtual delete)
func (service *Guest) Delete(guest *model.Guest, note string) error {

	// Delete this Guest
	if err := service.collection.Delete(guest, note); err != nil {
		return derp.Wrap(err, "service.Guest.Delete", "Error deleting Guest", guest, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Guest) ObjectType() string {
	return "Guest"
}

// New returns a fully initialized model.Guest as a data.Object.
func (service *Guest) ObjectNew() data.Object {
	result := model.NewGuest()
	return &result
}

func (service *Guest) ObjectID(object data.Object) primitive.ObjectID {

	if mention, ok := object.(*model.Guest); ok {
		return mention.GuestID
	}

	return primitive.NilObjectID
}

func (service *Guest) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Guest) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewGuest()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Guest) ObjectSave(object data.Object, comment string) error {
	if guest, ok := object.(*model.Guest); ok {
		return service.Save(guest, comment)
	}
	return derp.InternalError("service.Guest.ObjectSave", "Invalid Object Type", object)
}

func (service *Guest) ObjectDelete(object data.Object, comment string) error {
	if guest, ok := object.(*model.Guest); ok {
		return service.Delete(guest, comment)
	}
	return derp.InternalError("service.Guest.ObjectDelete", "Invalid Object Type", object)
}

func (service *Guest) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Guest.ObjectUserCan", "Not Authorized")
}

func (service *Guest) Schema() schema.Schema {
	return schema.New(model.GuestSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// LoadByEmail retrieves a single Guest from the database using the provided email address
func (service *Guest) LoadByEmail(emailAddress string, guest *model.Guest) error {
	criteria := exp.Equal("emailAddress", emailAddress)
	return service.Load(criteria, guest)
}

// LoadOrCreateByEmail searches for a Guest with the provided emailAddress.
// If a matching record is found, it updates the record with the new values (if necessary).
// If no matching record is found, it creates a new record with the provided values.
func (service *Guest) LoadOrCreate(emailAddress string, merchantAccountType string, remoteID string) (model.Guest, error) {

	// Try to load the guest using their email address
	guest := model.NewGuest()
	if err := service.LoadByEmail(emailAddress, &guest); !derp.IsNilOrNotFound(err) {
		return guest, derp.Wrap(err, "service.Guest.LoadOrCreateByEmail", "Error loading guest by email", emailAddress)
	}

	// Update the email and remoteID for the guest.  If changed, then save the record.
	if updated := guest.Update(emailAddress, merchantAccountType, remoteID); updated {
		if err := service.Save(&guest, "Updated"); err != nil {
			return guest, derp.Wrap(err, "service.Guest.LoadOrCreateByEmail", "Error saving guest", guest)
		}
	}

	// Done.
	return guest, nil
}
