package service

import (
	"math/rand"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/rosetta/list"
	"github.com/davecgh/go-spew/spew"
	"github.com/mmcdole/gofeed"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Subscription manages all interactions with the Subscription collection
type Subscription struct {
	collection     data.Collection
	streamService  *Stream
	contentService *Content
	queue          *queue.Queue
	closed         chan bool
}

// NewSubscription returns a fully populated Subscription service.
func NewSubscription(collection data.Collection, streamService *Stream, contentService *Content, queue *queue.Queue) Subscription {

	service := Subscription{
		collection:     collection,
		streamService:  streamService,
		contentService: contentService,
		queue:          queue,
		closed:         make(chan bool),
	}

	service.Refresh(collection)

	return service
}

/*******************************************
 * Lifecycle Methods
 *******************************************/

// Refresh updates any stateful data that is cached inside this service.
func (service *Subscription) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops the subscription service watcher
func (service *Subscription) Close() {
	close(service.closed)
}

// Start begins the background scheduler that checks each subscription
// according to its own polling frequency
// TODO: Need to make this configurable on a per-metal basis so that
// clusters can work together without hammering the Subscription collection.
func (service *Subscription) Start() {

	rand.Seed(time.Now().UnixNano())

	// query the database every minute, looking for subscriptions that should be loaded from the web.
	for {

		// Poll randomly between 1 and 5 minutes
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)

		// If (for some reason) the service collection is still nil, then
		// wait this one out.
		if service.collection == nil {
			continue
		}

		// Get a list of all subscriptions that can be polled
		it, err := service.ListPollable()

		if err != nil {
			derp.Report(derp.Wrap(err, "service.Subscription.Run", "Error listing pollable subscriptions"))
			continue
		}

		subscription := model.NewSubscription()

		for it.Next(&subscription) {
			select {

			// If we're done, we're done.
			case <-service.closed:
				return

			// Check each subscription one at a time (this may be slow but its okay)
			default:
				service.CheckSubscription(&subscription)
				subscription = model.NewSubscription()
			}
		}
	}
}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Subscription that is ready to use
func (service *Subscription) New() model.Subscription {
	return model.NewSubscription()
}

// List returns an iterator containing all of the Subscriptions who match the provided criteria
func (service *Subscription) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Subscription from the database
func (service *Subscription) Load(criteria exp.Expression, result *model.Subscription) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Subscription", "Error loading Subscription", criteria)
	}

	return nil
}

// Save adds/updates an Subscription in the database
func (service *Subscription) Save(subscription *model.Subscription, note string) error {

	if err := service.collection.Save(subscription, note); err != nil {
		return derp.Wrap(err, "service.Subscription", "Error saving Subscription", subscription, note)
	}

	return nil
}

// Delete removes an Subscription from the database (virtual delete)
func (service *Subscription) Delete(subscription *model.Subscription, note string) error {

	if err := service.collection.Delete(subscription, note); err != nil {
		return derp.Wrap(err, "service.Subscription", "Error deleting Subscription", subscription, note)
	}

	return nil
}

/*******************************************
 * Custom Queries
 *******************************************/

func (service *Subscription) ListPollable() (data.Iterator, error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix())
	return service.List(criteria, option.SortAsc("lastPolled"))
}

func (service *Subscription) ListByUserID(userID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID)
	return service.List(criteria, option.SortAsc("lastPolled"))
}

func (service *Subscription) LoadByID(subscriptionID primitive.ObjectID, result *model.Subscription) error {

	criteria := exp.Equal("_id", subscriptionID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Subscription.LoadByID", "Error loading Subscription", criteria)
	}

	return nil
}

/*******************************************
 * Subscription Methods
 *******************************************/

// CheckSubscriptions parses the RSS feed and adds/updates a new stream for each item in it.
func (service *Subscription) CheckSubscription(subscription *model.Subscription) {

	const location = "service.Subscription.PollSubscription"

	spew.Dump("Polling: ", subscription)

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(subscription.URL)

	// TODO: Limit retries on failed subscription URLs

	if err != nil {
		derp.Report(derp.Wrap(err, location, "Error Parsing Feed URL"))
		return
	}

	// Update each stream in the RSS feed
	for _, item := range feed.Items {
		if err := service.updateStream(subscription, item); err != nil {
			derp.Report(derp.Wrap(err, location, "Error updating local stream"))
		}
	}

	// Mark the subscription as having been polled
	subscription.MarkPolled()

	// Try to save the updated subscription to the database
	if err := service.Save(subscription, "Polling Subscription"); err != nil {
		derp.Report(derp.Wrap(err, location, "Error saving subscription"))
	}
}

// updateStream adds/updates an individual stream based on an RSS item
func (service *Subscription) updateStream(sub *model.Subscription, item *gofeed.Item) error {

	const location = "service.Subscription.updateStream"

	stream := model.NewStream()

	if err := service.streamService.LoadByOriginURL(sub.ParentStreamID, item.Link, &stream); err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading local stream")
		}

		// Fall through means "not found" which means "make a new stream"
		stream = model.NewStream()
		stream.TemplateID = "user-inbox-item"
		stream.ParentID = sub.ParentStreamID
		stream.Origin.URL = item.Link
		stream.StateID = "unread"
	}

	// Calculate update date.
	updateDate := item.PublishedParsed.Unix()

	if item.UpdatedParsed != nil {
		updateDate = item.UpdatedParsed.Unix()
	}

	// If stream has been updated since previous save, then set new values and update
	if stream.Origin.UpdateDate > updateDate {

		// Populate stream header and content
		stream.Label = item.Title
		stream.Description = item.Description
		stream.Content = service.contentService.New("HTML", item.Content)
		stream.PublishDate = item.PublishedParsed.Unix()
		stream.Origin.UpdateDate = updateDate
		stream.Tags = sub.Tags

		// Reset Author
		stream.Author = model.NewAuthorLink()
		if item.Author != nil {
			stream.Author.Name = item.Author.Name
			stream.Author.EmailAddress = item.Author.Email
		}

		// Reset Image
		if item.Image != nil {
			stream.ThumbnailImage = item.Image.URL
		} else {
			stream.ThumbnailImage = ""

			// Search for an image in the enclosures
			for _, enclosure := range item.Enclosures {
				if list.Slash(enclosure.Type).Head() == "image" {
					stream.ThumbnailImage = enclosure.URL
					break
				}
			}
		}

		// Try to save the new/updated stream
		if err := service.streamService.Save(&stream, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "service.Subscription.Poll", "Error saving stream")
		}
	}

	return nil
}
