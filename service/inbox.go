package service

import (
	"time"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/datatype"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/whisperverse/activitystream/reader"
	"github.com/whisperverse/whisperverse/model"
)

type Inbox struct {
	collection    data.Collection
	streamService *Stream
	userService   *User
}

func NewInbox(collection data.Collection, streamService *Stream, userService *User) Inbox {
	return Inbox{
		collection:    collection,
		streamService: streamService,
		userService:   userService,
	}
}

/*******************************************
 * COMMON DATA FUNCTIONS
 *******************************************/

// List returns an iterator containing all of the Inboxes who match the provided criteria
func (service *Inbox) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Inbox from the database
func (service *Inbox) Load(criteria exp.Expression, result *model.InboxItem) error {
	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Inbox.Load", "Error loading Inbox", criteria)
	}

	return nil
}

// Save adds/updates an Inbox in the database
func (service *Inbox) Save(user *model.InboxItem, note string) error {

	if err := service.collection.Save(user, note); err != nil {
		return derp.Wrap(err, "service.Inbox.Save", "Error saving Inbox", user, note)
	}

	return nil
}

// Delete removes an Inbox from the database (virtual delete)
func (service *Inbox) Delete(user *model.InboxItem, note string) error {

	if err := service.collection.Delete(user, note); err != nil {
		return derp.Wrap(err, "service.Inbox.Delete", "Error deleting Inbox", user, note)
	}

	return nil
}

/*******************************************
 * GENERIC DATA FUNCTIONS
 *******************************************/

// New returns a fully initialized model.Stream as a data.Object.
func (service *Inbox) ObjectNew() data.Object {
	result := model.NewInboxItem()
	return &result
}

func (service *Inbox) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Inbox) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewInboxItem()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Inbox) ObjectSave(object data.Object, comment string) error {
	return service.Save(object.(*model.InboxItem), comment)
}

func (service *Inbox) ObjectDelete(object data.Object, comment string) error {
	return service.Delete(object.(*model.InboxItem), comment)
}

func (service *Inbox) Debug() datatype.Map {
	return datatype.Map{
		"service": "Inbox",
	}
}

/*******************************************
 * INBOX METHODS
 *******************************************/

func (service *Inbox) Receive(user *model.User, message map[string]interface{}) error {

	const location = "service.Inbox.Receive"

	// Parse the message into an ActivityPub object
	object := reader.New(message)
	stream := model.NewStream()

	// stream info
	stream.TemplateID = "social-inbox-item"
	stream.Label = object.Name()
	stream.Description = object.Summary()
	stream.ThumbnailImage = object.Icon()

	if publishDate := object.Published().UnixMilli(); publishDate > 0 {
		stream.PublishDate = publishDate
	} else {
		stream.PublishDate = time.Now().UnixMilli()
	}

	// actor info
	actor := object.ActorObject()
	stream.AuthorURL = actor.ID()
	stream.AuthorName = actor.Name()
	stream.AuthorImage = actor.Icon()

	// other activityPub info
	stream.Data["original"] = object

	// recipient info
	stream.ParentID = user.InboxID
	stream.Criteria.OwnerID = user.UserID

	if err := service.streamService.Save(&stream, "Imported from ActivityPub"); err != nil {
		return derp.Wrap(err, location, "error saving message", message)
	}

	return nil
}

// FindUsers locates all local users who are listed as recipients of the message.
func (service *Inbox) FindUsers(object reader.Object) ([]model.User, error) {

	result := make([]model.User, 0)
	identities := object.AllRecipients()
	iterator, err := service.userService.ListByIdentities(identities)

	if err != nil {
		return result, derp.Wrap(err, "whisperverse.service.Inbox.FindUsers", "Error querying users by identities", identities)
	}

	// Copy users from the iterator into a slice.
	user := model.NewUser()
	for iterator.Next(&user) {
		result = append(result, user.Copy())
		user = model.NewUser()
	}

	return result, nil
}
