package service

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/form"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder manages all interactions with a user's Folder
type Folder struct {
	collection data.Collection
}

// NewFolder returns a fully populated Folder service
func NewFolder(collection data.Collection) Folder {
	service := Folder{
		collection: collection,
	}

	service.Refresh(collection)
	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Folder) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops any background processes controlled by this service
func (service *Folder) Close() {

}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Folder that is ready to use
func (service *Folder) New() model.Folder {
	return model.NewFolder()
}

// Query returns a slice of Folders that math the provided criteria
func (service *Folder) Query(criteria exp.Expression, options ...option.Option) ([]model.Folder, error) {
	result := []model.Folder{}
	err := service.collection.Query(&result, criteria, options...)
	return result, err
}

// List returns an iterator containing all of the Folders that match the provided criteria
func (service *Folder) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Folder from the database
func (service *Folder) Load(criteria exp.Expression, result *model.Folder) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Report(derp.Wrap(err, "service.Folder.Load", "Error loading Folder", criteria))
	}

	return nil
}

// Save adds/updates an Folder in the database
func (service *Folder) Save(folder *model.Folder, note string) error {

	// Clean the value before saving
	if err := service.Schema().Clean(folder); err != nil {
		return derp.Wrap(err, "service.Folder.Save", "Error cleaning Folder", folder)
	}

	// Save the value to the database
	if err := service.collection.Save(folder, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error saving Folder", folder, note)
	}

	return nil
}

// Delete removes an Folder from the database (virtual delete)
func (service *Folder) Delete(inboxItem *model.Folder, note string) error {

	// Delete Folder record last.
	if err := service.collection.Delete(inboxItem, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error deleting Folder", inboxItem, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Folder) ObjectType() string {
	return "Folder"
}

// New returns a fully initialized model.Group as a data.Object.
func (service *Folder) ObjectNew() data.Object {
	result := model.NewFolder()
	return &result
}

func (service *Folder) ObjectID(object data.Object) primitive.ObjectID {

	if folder, ok := object.(*model.Folder); ok {
		return folder.FolderID
	}

	return primitive.NilObjectID
}

func (service *Folder) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, notDeleted(criteria), options...)
}

func (service *Folder) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Folder) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFolder()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Folder) ObjectSave(object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Save(folder, comment)
	}
	return derp.NewInternalError("service.Folder.ObjectSave", "Invalid object type", object)
}

func (service *Folder) ObjectDelete(object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Delete(folder, comment)
	}
	return derp.NewInternalError("service.Folder.ObjectDelete", "Invalid object type", object)
}

func (service *Folder) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Folder", "Not Authorized")
}

func (service *Folder) Schema() schema.Schema {
	return schema.New(model.FolderSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Folder) QueryByUserID(userID primitive.ObjectID) ([]model.Folder, error) {
	return service.Query(exp.Equal("userId", userID), option.SortAsc("label"))
}

// LoadByToken locates a single stream that matches the provided token
func (service *Folder) LoadByToken(userID primitive.ObjectID, token string, result *model.Folder) error {

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {

		criteria := exp.And(
			exp.Equal("_id", folderID),
			exp.Equal("userId", userID),
		)

		return service.Load(criteria, result)
	}

	return derp.NewBadRequestError("service.Folder", "Invalid token", token)
}

// LoadBySource locates a single stream that matches the provided OriginURL
func (service *Folder) LoadByOriginURL(userID primitive.ObjectID, originURL string, result *model.Folder) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("origin.url", originURL)

	return service.Load(criteria, result)
}

/*******************************************
 * Misc Actions
 *******************************************/

func (service *Folder) LookupProvider(userID primitive.ObjectID) form.LookupProvider {
	return NewFolderLookupProvider(service, userID)
}
