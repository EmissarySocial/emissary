package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/maps"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// InboxFolder manages all interactions with a user's InboxFolder
type InboxFolder struct {
	collection data.Collection
}

// NewInboxFolder returns a fully populated InboxFolder service
func NewInboxFolder(collection data.Collection) InboxFolder {
	service := InboxFolder{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *InboxFolder) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *InboxFolder) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized InboxFolder that is ready to use
func (service *InboxFolder) New() model.InboxFolder {
	return model.NewInboxFolder()
}

// Query returns a slice of InboxFolders that math the provided criteria
func (service *InboxFolder) Query(criteria exp.Expression, options ...option.Option) ([]model.InboxFolder, error) {
	result := []model.InboxFolder{}
	err := service.collection.Query(&result, criteria, options...)
	return result, err
}

// List returns an iterator containing all of the InboxFolders that match the provided criteria
func (service *InboxFolder) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an InboxFolder from the database
func (service *InboxFolder) Load(criteria exp.Expression, result *model.InboxFolder) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Report(derp.Wrap(err, "service.InboxFolder.Load", "Error loading InboxFolder", criteria))
	}

	return nil
}

// Save adds/updates an InboxFolder in the database
func (service *InboxFolder) Save(inboxFolder *model.InboxFolder, note string) error {

	if err := service.collection.Save(inboxFolder, note); err != nil {
		return derp.Wrap(err, "service.InboxFolder", "Error saving InboxFolder", inboxFolder, note)
	}

	return nil
}

// Delete removes an InboxFolder from the database (virtual delete)
func (service *InboxFolder) Delete(inboxItem *model.InboxFolder, note string) error {

	// Delete InboxFolder record last.
	if err := service.collection.Delete(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.InboxFolder", "Error deleting InboxFolder", inboxItem, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// New returns a fully initialized model.Group as a data.Object.
func (service *InboxFolder) ObjectNew() data.Object {
	result := model.NewInboxFolder()
	return &result
}

func (service *InboxFolder) ObjectID(object data.Object) primitive.ObjectID {

	if inboxFolder, ok := object.(*model.InboxFolder); ok {
		return inboxFolder.InboxFolderID
	}

	return primitive.NilObjectID
}
func (service *InboxFolder) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *InboxFolder) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewInboxFolder()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *InboxFolder) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.InboxFolder), comment)
}

func (service *InboxFolder) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.InboxFolder), comment)
}

func (service *InboxFolder) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.InboxFolder", "Not Authorized")
}

func (service *InboxFolder) Schema() schema.Element {
	return model.InboxFolderSchema()
}

func (service *InboxFolder) Debug() maps.Map {
	return maps.Map{
		"service": "InboxFolder",
	}
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *InboxFolder) QueryByUserID(userID primitive.ObjectID) ([]model.InboxFolder, error) {
	return service.Query(exp.Equal("userId", userID), option.SortAsc("rank"))
}

// LoadByToken locates a single stream that matches the provided token
func (service *InboxFolder) LoadByToken(userID primitive.ObjectID, token string, result *model.InboxFolder) error {

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {

		criteria := exp.And(
			exp.Equal("_id", folderID),
			exp.Equal("userId", userID),
		)

		return service.Load(criteria, result)
	}

	return derp.NewBadRequestError("service.InboxFolder", "Invalid token", token)
}

// LoadBySource locates a single stream that matches the provided OriginURL
func (service *InboxFolder) LoadByOriginURL(userID primitive.ObjectID, originURL string, result *model.InboxFolder) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("origin.url", originURL)

	return service.Load(criteria, result)
}
