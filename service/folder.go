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
	domainService    *Domain
	followingService *Following
	inboxService     *Inbox
	themeService     *Theme
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
func (service *Folder) Refresh(domainService *Domain, followingService *Following, inboxService *Inbox, themeService *Theme) {
	service.domainService = domainService
	service.followingService = followingService
	service.inboxService = inboxService
	service.themeService = themeService
}

// Close stops any background processes controlled by this service
func (service *Folder) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

func (service *Folder) collection(session data.Session) data.Collection {
	return session.Collection("Folder")
}

// New creates a newly initialized Folder that is ready to use
func (service *Folder) New() model.Folder {
	return model.NewFolder()
}

// Count returns the number of records that match the provided criteria
func (service *Folder) Count(session data.Session, criteria exp.Expression) (int64, error) {
	return service.collection(session).Count(notDeleted(criteria))
}

// Query returns a slice of Folders that math the provided criteria
func (service *Folder) Query(session data.Session, criteria exp.Expression, options ...option.Option) ([]model.Folder, error) {
	result := []model.Folder{}
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// Range returns an iterator containing all of the Folders that match the provided criteria
func (service *Folder) Range(session data.Session, criteria exp.Expression, options ...option.Option) (iter.Seq[model.Folder], error) {

	const location = "service.Folder.Range"

	iter, err := service.List(session, criteria, options...)

	if err != nil {
		return nil, derp.Wrap(err, location, "Unable to create iterator", criteria)
	}

	return RangeFunc(iter, model.NewFolder), nil
}

// List returns an iterator containing all of the Folders that match the provided criteria
func (service *Folder) List(session data.Session, criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection(session).Iterator(notDeleted(criteria), options...)
}

// Load retrieves an Folder from the database
func (service *Folder) Load(session data.Session, criteria exp.Expression, result *model.Folder) error {

	const location = "service.Folder.Load"

	if err := service.collection(session).Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, location, "Unable to load Folder", criteria)
	}

	return nil
}

// Save adds/updates an Folder in the database
func (service *Folder) Save(session data.Session, folder *model.Folder, comment string) error {

	const location = "service.Folder.Save"

	// Validate the value before saving
	if err := service.Schema().Validate(folder); err != nil {
		return derp.Wrap(err, location, "Invalid Folder data", folder)
	}

	// Save the value to the database
	if err := service.collection(session).Save(folder, comment); err != nil {
		return derp.Wrap(err, location, "Unable to save Folder", folder, comment)
	}

	return nil
}

// Delete removes an Folder from the database (virtual delete)
func (service *Folder) Delete(session data.Session, folder *model.Folder, comment string) error {

	const location = "service.Folder.Delete"

	// Delete the folder
	if err := service.collection(session).Delete(folder, comment); err != nil {
		return derp.Wrap(err, location, "Unable to delete Folder", folder, comment)
	}

	// Delete inbox items
	if err := service.inboxService.DeleteByFolder(session, folder.UserID, folder.FolderID); err != nil {
		return derp.Wrap(err, location, "Unable to delete related `Inbox Message` records.", folder, comment)
	}

	// Delete any followings
	if err := service.followingService.DeleteByFolder(session, folder.UserID, folder.FolderID, comment); err != nil {
		return derp.Wrap(err, location, "Unable to delete related `Following` records.")
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

func (service *Folder) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

func (service *Folder) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewFolder()
	err := service.Load(session, criteria, &result)
	return &result, err
}

func (service *Folder) ObjectSave(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Save(session, folder, comment)
	}
	return derp.InternalError("service.Folder.ObjectSave", "Invalid object type", object)
}

func (service *Folder) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Delete(session, folder, comment)
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
func (service *Folder) RangeByUserID(session data.Session, userID primitive.ObjectID) (iter.Seq[model.Folder], error) {
	return service.Range(session, exp.Equal("userId", userID), option.SortAsc("rank"))
}

// DeleteByUserID removes all folders for a given user
func (service *Folder) DeleteByUserID(session data.Session, userID primitive.ObjectID, comment string) error {

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, "service.Folder.DeleteByUserID", "Unable to list folders", userID)
	}

	for folder := range rangeFunc {
		if err := service.Delete(session, &folder, comment); err != nil {
			return derp.Wrap(err, "service.Folder.DeleteByUserID", "Unable to delete folder", folder)
		}
	}

	return nil
}

// QueryByUserID returns all folders for a given user
func (service *Folder) QueryByUserID(session data.Session, userID primitive.ObjectID) ([]model.Folder, error) {
	return service.Query(session, exp.Equal("userId", userID), option.SortAsc("rank"))
}

// LoadByID loads a single stream that matches the provided ID
func (service *Folder) LoadByID(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, result *model.Folder) error {

	criteria := exp.
		Equal("_id", folderID).
		AndEqual("userId", userID)

	return service.Load(session, criteria, result)
}

// LoadByToken loads a single stream that matches the provided token
func (service *Folder) LoadByToken(session data.Session, userID primitive.ObjectID, token string, result *model.Folder) error {

	if folderID, err := primitive.ObjectIDFromHex(token); err == nil {

		criteria := exp.And(
			exp.Equal("_id", folderID),
			exp.Equal("userId", userID),
		)

		return service.Load(session, criteria, result)
	}

	return derp.BadRequestError("service.Folder", "Invalid token", token)
}

// LoadByLabel loads a single stream that matches the provided label
func (service *Folder) LoadByLabel(session data.Session, userID primitive.ObjectID, label string, result *model.Folder) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("label", label)

	return service.Load(session, criteria, result)
}

/******************************************
 * Other Behaviors
 ******************************************/

// CalculateUnreadCount counts the number of items in a folder that were created AFTER the provided minRank,
// then updates the folder's "unreadCount" and "readDate" fields
func (service *Folder) CalculateUnreadCount(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) error {

	const location = "service.Folder.CalculateUnreadCount"

	if userID.IsZero() {
		return derp.BadRequestError(location, "UserID cannot be empty", userID)
	}

	if folderID.IsZero() {
		return derp.BadRequestError(location, "FolderID cannot be empty", folderID)
	}

	unreadCount, err := service.inboxService.CountUnreadMessages(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, location, "Unable to count unread messages", userID, folderID)
	}

	collection := service.collection(session)

	if err := queries.FolderSetUnreadCount(collection, userID, folderID, unreadCount); err != nil {
		return derp.Wrap(err, "service.Folder", "Unable to update folder read date", userID, folderID)
	}

	return nil
}

func (service *Folder) CreateDefaultFolders(session data.Session, userID primitive.ObjectID) error {

	domain := service.domainService.Get()
	theme := service.themeService.GetTheme(domain.ThemeID)

	for index, data := range theme.DefaultFolders {
		folder := model.NewFolder()
		folder.UserID = userID
		folder.Rank = index
		folder.Label = data.GetString("label")
		folder.Layout = first.String(data.GetString("layout"), model.FolderLayoutSocial)
		folder.Icon = first.String(data.GetString("icon"), "folder")

		if err := service.Save(session, &folder, "Create default folder"); err != nil {
			return err
		}
	}

	return nil
}
