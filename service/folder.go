package service

import (
	"iter"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queries"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/schema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder manages all interactions with a user's Folder
type Folder struct {
	collection    data.Collection
	themeService  *Theme
	domainService *Domain
	inboxService  *Inbox
}

// NewFolder returns a fully populated Folder service
func NewFolder() Folder {
	service := Folder{}
	return service
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Folder) Refresh(collection data.Collection, themeService *Theme, domainService *Domain, inboxService *Inbox) {
	service.collection = collection
	service.themeService = themeService
	service.domainService = domainService
	service.inboxService = inboxService
}

// Close stops any background processes controlled by this service
func (service *Folder) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// New creates a newly initialized Folder that is ready to use
func (service *Folder) New() model.Folder {
	return model.NewFolder()
}

// Count returns the number of records that match the provided criteria
func (service *Folder) Count(criteria exp.Expression) (int64, error) {
	return service.collection.Count(notDeleted(criteria))
}

// Query returns a slice of Folders that math the provided criteria
func (service *Folder) Query(criteria exp.Expression, options ...option.Option) ([]model.Folder, error) {
	result := []model.Folder{}
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Range returns an iterator containing all of the Folders that match the provided criteria
func (service *Folder) Range(criteria exp.Expression, options ...option.Option) (iter.Seq[model.Folder], error) {

	iter, err := service.List(criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, "service.Folder.Range", "Error creating iterator", criteria)
	}

	return RangeFunc(iter, model.NewFolder), nil
}

// List returns an iterator containing all of the Folders that match the provided criteria
func (service *Folder) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Folder from the database
func (service *Folder) Load(criteria exp.Expression, result *model.Folder) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Folder.Load", "Error loading Folder", criteria)
	}

	return nil
}

// Save adds/updates an Folder in the database
func (service *Folder) Save(folder *model.Folder, note string) error {

	// Validate the value before saving
	if err := service.Schema().Validate(folder); err != nil {
		return derp.Wrap(err, "service.Folder.Save", "Error validating Folder", folder)
	}

	// Save the value to the database
	if err := service.collection.Save(folder, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error saving Folder", folder, note)
	}

	return nil
}

// Delete removes an Folder from the database (virtual delete)
func (service *Folder) Delete(folder *model.Folder, note string) error {

	if err := service.inboxService.DeleteByFolder(folder.UserID, folder.FolderID); err != nil {
		return derp.Wrap(err, "service.Folder", "Error deleting Folder activities", folder, note)
	}

	// Delete Folder record last.
	if err := service.collection.Delete(folder, note); err != nil {
		return derp.Wrap(err, "service.Folder", "Error deleting Folder", folder, note)
	}

	return nil
}

/******************************************
 * Model Service Methods
 ******************************************/

// ObjectType returns the type of object that this service manages
func (service *Folder) ObjectType() string {
	return "Folder"
}

// New returns a fully initialized model.Folder as a data.Object.
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

func (service *Folder) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFolder()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Folder) ObjectSave(object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Save(folder, comment)
	}
	return derp.InternalError("service.Folder.ObjectSave", "Invalid object type", object)
}

func (service *Folder) ObjectDelete(object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Delete(folder, comment)
	}
	return derp.InternalError("service.Folder.ObjectDelete", "Invalid object type", object)
}

func (service *Folder) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.UnauthorizedError("service.Folder", "Not Authorized")
}

func (service *Folder) Schema() schema.Schema {
	return schema.New(model.FolderSchema())
}

/******************************************
 * Custom Queries
 ******************************************/

// RangeByUserID returns an iterator containing all of the Folders for a given user
func (service *Folder) RangeByUserID(userID primitive.ObjectID) (iter.Seq[model.Folder], error) {
	return service.Range(exp.Equal("userId", userID), option.SortAsc("rank"))
}

// DeleteByUserID removes all folders for a given user
func (service *Folder) DeleteByUserID(userID primitive.ObjectID, comment string) error {

	rangeFunc, err := service.RangeByUserID(userID)

	if err != nil {
		return derp.Wrap(err, "service.Folder.DeleteByUserID", "Error listing folders", userID)
	}

	for folder := range rangeFunc {
		if err := service.Delete(&folder, comment); err != nil {
			return derp.Wrap(err, "service.Folder.DeleteByUserID", "Error deleting folder", folder)
		}
	}

	return nil
}

// QueryByUserID returns all folders for a given user
func (service *Folder) QueryByUserID(userID primitive.ObjectID) ([]model.Folder, error) {
	return service.Query(exp.Equal("userId", userID), option.SortAsc("rank"))
}

// LoadByID loads a single stream that matches the provided ID
func (service *Folder) LoadByID(userID primitive.ObjectID, folderID primitive.ObjectID, result *model.Folder) error {

	criteria := exp.
		Equal("_id", folderID).
		AndEqual("userId", userID)

	return service.Load(criteria, result)
}

// LoadByToken loads a single stream that matches the provided token
func (service *Folder) LoadByToken(userID primitive.ObjectID, token string, result *model.Folder) error {

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {

		criteria := exp.And(
			exp.Equal("_id", folderID),
			exp.Equal("userId", userID),
		)

		return service.Load(criteria, result)
	}

	return derp.BadRequestError("service.Folder", "Invalid token", token)
}

// LoadByLabel loads a single stream that matches the provided label
func (service *Folder) LoadByLabel(userID primitive.ObjectID, label string, result *model.Folder) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("label", label)

	return service.Load(criteria, result)
}

/******************************************
 * Other Behaviors
 ******************************************/

func (service *Folder) ReCalculateUnreadCountFromFolder(userID primitive.ObjectID, folderID primitive.ObjectID) error {

	const location = "service.Folder.ReCalculateUnreadCountFromFolder"

	if userID.IsZero() {
		return derp.BadRequestError(location, "UserID cannot be empty", userID)
	}

	if folderID.IsZero() {
		return derp.BadRequestError(location, "FolderID cannot be empty", folderID)
	}

	// Try to load the folder
	folder := model.NewFolder()
	if err := service.LoadByID(userID, folderID, &folder); err != nil {
		return derp.Wrap(err, location, "Unable to load Folder")
	}

	// Recalculate unread counts
	if err := service.CalculateUnreadCount(userID, folderID); err != nil {
		return derp.Wrap(err, location, "Unable to update `Unread` count")
	}

	return nil
}

// CalculateUnreadCount counts the number of items in a folder that were created AFTER the provided minRank,
// then updates the folder's "unreadCount" and "readDate" fields
func (service *Folder) CalculateUnreadCount(userID primitive.ObjectID, folderID primitive.ObjectID) error {

	unreadCount, err := service.inboxService.CountUnreadMessages(userID, folderID)

	if err != nil {
		return derp.Wrap(err, "service.Folder.CalculateUnreadCount", "Error counting unread messages", userID, folderID)
	}

	return service.SetUnreadCount(userID, folderID, unreadCount)
}

// SetUnreadCount uses an optimized query to update the the "readDate" and "unreadCount" of a particular folder
func (service *Folder) SetUnreadCount(userID primitive.ObjectID, folderID primitive.ObjectID, unreadCount int) error {

	if err := queries.FolderSetUnreadCount(service.collection, userID, folderID, unreadCount); err != nil {
		return derp.Wrap(err, "service.Folder", "Error updating folder read date", userID, folderID)
	}

	return nil
}

func (service *Folder) CreateDefaultFolders(userID primitive.ObjectID) error {

	domain := service.domainService.Get()
	theme := service.themeService.GetTheme(domain.ThemeID)

	for index, data := range theme.DefaultFolders {
		folder := model.NewFolder()
		folder.UserID = userID
		folder.Rank = index
		folder.Label = data.GetString("label")
		folder.Layout = first.String(data.GetString("layout"), model.FolderLayoutNewspaper)
		folder.Icon = first.String(data.GetString("icon"), "folder")

		if err := service.Save(&folder, "Create default folder"); err != nil {
			return err
		}
	}

	return nil
}
