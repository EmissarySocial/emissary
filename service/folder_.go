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
	"github.com/benpate/rosetta/sliceof"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Folder manages all interactions with a user's Folder
type Folder struct {
	domainService     *Domain
	followingService  *Following
	importItemService *ImportItem
	newsFeedService   *NewsFeed
	themeService      *Theme
}

// NewFolder returns a fully populated Folder service
func NewFolder() Folder {
	return Folder{}
}

/******************************************
 * Lifecycle Methods
 ******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Folder) Refresh(factory *Factory) {
	service.domainService = factory.Domain()
	service.followingService = factory.Following()
	service.importItemService = factory.ImportItem()
	service.newsFeedService = factory.NewsFeed()
	service.themeService = factory.Theme()
}

// Close stops any background processes controlled by this service
func (service *Folder) Close() {

}

/******************************************
 * Common Data Methods
 ******************************************/

// collection returns the Folder collection for the provided database session
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
		return nil, derp.Wrap(err, location, "Creating iterator", criteria)
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
		return derp.Wrap(err, location, "Loading Folder", criteria)
	}

	return nil
}

// Save adds/updates an Folder in the database
func (service *Folder) Save(session data.Session, folder *model.Folder, comment string) error {

	const location = "service.Folder.Save"

	// Validate the value before saving
	if _, err := service.Schema().Validate(folder); err != nil {
		return derp.Wrap(err, location, "Invalid Folder data", folder)
	}

	// RULE: The Label MUST NOT collide with this User's other Folders
	if err := service.ValidateLabel(session, folder.UserID, folder.FolderID, folder.Label); err != nil {
		return derp.Wrap(err, location, "Invalid Folder name", folder)
	}

	// Save the value to the database
	if err := service.collection(session).Save(folder, comment); err != nil {
		return derp.Wrap(err, location, "Saving Folder", folder, comment)
	}

	return nil
}

// Delete removes an Folder from the database (virtual delete)
func (service *Folder) Delete(session data.Session, folder *model.Folder, comment string) error {

	const location = "service.Folder.Delete"

	// Delete the folder
	if err := service.collection(session).Delete(folder, comment); err != nil {
		return derp.Wrap(err, location, "Deleting Folder", folder, comment)
	}

	// Delete inbox items
	if err := service.newsFeedService.DeleteByFolder(session, folder.UserID, folder.FolderID); err != nil {
		return derp.Wrap(err, location, "Deleting related `NewsFeed Message` records.", folder, comment)
	}

	// Delete any followings
	if err := service.followingService.DeleteByFolder(session, folder.UserID, folder.FolderID, comment); err != nil {
		return derp.Wrap(err, location, "Deleting related `Following` records.")
	}

	return nil
}

/******************************************
 * Special Case Methods
 ******************************************/

// QueryIDOnly returns a slice of IDOnly records that match the provided criteria
func (service *Folder) QueryIDOnly(session data.Session, criteria exp.Expression, options ...option.Option) (sliceof.Object[model.IDOnly], error) {
	result := make([]model.IDOnly, 0)
	options = append(options, option.Fields("_id"))
	err := service.collection(session).Query(&result, notDeleted(criteria), options...)
	return result, err
}

// HardDeleteByID removes a specific Folder record, without applying any additional business rules
func (service *Folder) HardDeleteByID(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) error {

	const location = "service.Folder.HardDeleteByID"

	criteria := exp.Equal("userId", userID).AndEqual("_id", folderID)

	if err := service.collection(session).HardDelete(criteria); err != nil {
		return derp.Wrap(err, location, "Deleting Folder", "userID: "+userID.Hex(), "folderID: "+folderID.Hex())
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

// ObjectID returns the unique ID of the provided Folder. Implements the ModelService interface.
func (service *Folder) ObjectID(object data.Object) primitive.ObjectID {

	if folder, ok := object.(*model.Folder); ok {
		return folder.FolderID
	}

	return primitive.NilObjectID
}

// ObjectQuery returns every Folder that matches the provided criteria. Implements the ModelService interface.
func (service *Folder) ObjectQuery(session data.Session, result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection(session).Query(result, notDeleted(criteria), options...)
}

// ObjectLoad retrieves a single Folder as a data.Object. Implements the ModelService interface.
func (service *Folder) ObjectLoad(session data.Session, criteria exp.Expression) (data.Object, error) {
	result := model.NewFolder()
	err := service.Load(session, criteria, &result)
	return &result, err
}

// ObjectSave adds or updates a Folder in the database. Implements the ModelService interface.
func (service *Folder) ObjectSave(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Save(session, folder, comment)
	}
	return derp.Internal("service.Folder.ObjectSave", "Invalid object type", object)
}

// ObjectDelete marks a Folder as deleted. Implements the ModelService interface.
func (service *Folder) ObjectDelete(session data.Session, object data.Object, comment string) error {
	if folder, ok := object.(*model.Folder); ok {
		return service.Delete(session, folder, comment)
	}
	return derp.Internal("service.Folder.ObjectDelete", "Invalid object type", object)
}

// ObjectUserCan reports whether the provided Authorization may run an action on a Folder. Implements the ModelService interface.
func (service *Folder) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.Unauthorized("service.Folder", "Not Authorized")
}

// Schema returns the rosetta schema that describes a Folder
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

	const location = "service.Folder.DeleteByUserID"

	rangeFunc, err := service.RangeByUserID(session, userID)

	if err != nil {
		return derp.Wrap(err, location, "Listing folders", userID)
	}

	for folder := range rangeFunc {
		if err := service.Delete(session, &folder, comment); err != nil {
			return derp.Wrap(err, location, "Deleting folder", folder)
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

	// Convert the token to an ObjectID
	folderID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.BadRequest("service.Folder", "Invalid token", token)
	}

	return service.LoadByID(session, userID, folderID, result)
}

// LoadByLabel loads a single Folder that matches the provided label
func (service *Folder) LoadByLabel(session data.Session, userID primitive.ObjectID, label string, result *model.Folder) error {

	criteria := exp.
		Equal("userId", userID).
		AndEqual("label", label)

	return service.Load(session, criteria, result)
}

/******************************************
 * Other Behaviors
 ******************************************/

// ValidateLabel returns an error if a Label cannot be used by the provided Folder
func (service *Folder) ValidateLabel(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, label string) error {

	const location = "service.Folder.ValidateLabel"

	// RULE: Label is required
	if label == "" {
		return derp.BadRequest(location, "Name is required", label)
	}

	// RULE: Label must be unique within this User's Folders
	if service.LabelExists(session, userID, folderID, label) {
		return derp.BadRequest(location, "You already have a folder with this name", label)
	}

	return nil
}

// LabelExists returns TRUE if a Label is already in use by another one of this User's Folders
func (service *Folder) LabelExists(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID, label string) bool {

	criteria := exp.Equal("userId", userID).
		AndEqual("label", label).
		AndNotEqual("_id", folderID)

	// Any error (including "not found") means that the Label is available
	folder := model.NewFolder()
	return service.Load(session, criteria, &folder) == nil
}

// CalculateUnreadCount counts the number of items in a folder that were created AFTER the provided minRank,
// then updates the folder's "unreadCount" and "readDate" fields
func (service *Folder) CalculateUnreadCount(session data.Session, userID primitive.ObjectID, folderID primitive.ObjectID) error {

	const location = "service.Folder.CalculateUnreadCount"

	if userID.IsZero() {
		return nil
	}

	if folderID.IsZero() {
		return nil
	}

	unreadCount, err := service.newsFeedService.CountUnreadNewsItems(session, userID, folderID)

	if err != nil {
		return derp.Wrap(err, location, "Counting unread messages", userID, folderID)
	}

	collection := service.collection(session)

	if err := queries.FolderSetUnreadCount(collection, userID, folderID, unreadCount); err != nil {
		return derp.Wrap(err, location, "Updating folder read date", userID, folderID)
	}

	return nil
}

// CreateDefaultFolders creates the starter Folders that a new User begins with, as defined by this Domain's Theme
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
