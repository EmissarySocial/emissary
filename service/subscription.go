package service

import (
	"bytes"
	"math/rand"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/PuerkitoBio/goquery"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/list"
	"github.com/mmcdole/gofeed"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/net/html"
)

// Subscription manages all interactions with the Subscription collection
type Subscription struct {
	collection   data.Collection
	inboxService *Inbox
	queue        *queue.Queue
	closed       chan bool
}

// NewSubscription returns a fully populated Subscription service.
func NewSubscription(collection data.Collection, inboxService *Inbox, queue *queue.Queue) Subscription {

	service := Subscription{
		collection:   collection,
		inboxService: inboxService,
		queue:        queue,
		closed:       make(chan bool),
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
		// time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
		time.Sleep(time.Duration(10 * time.Second))

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
				derp.Report(service.CheckSubscription(&subscription))
				subscription = model.NewSubscription()
			}
		}

		// Poll randomly between 1 and 5 minutes
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
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

// QueryByUserID returns a slice of all subscriptions for a given user
func (service *Subscription) QueryByUserID(userID primitive.ObjectID) ([]model.SubscriptionSummary, error) {
	result := make([]model.SubscriptionSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection.Query(&result, criteria)

	return result, err
}

// ListPollable returns an iterator of all subscriptions that are ready to be polled
func (service *Subscription) ListPollable() (data.Iterator, error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix())
	return service.List(criteria, option.SortAsc("lastPolled"))
}

// ListByUserID returns an iterator of all subscriptions for a given user
func (service *Subscription) ListByUserID(userID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID)
	return service.List(criteria, option.SortAsc("lastPolled"))
}

// LoadByID retrieves an Subscription from the database.  UserID is required to prevent
// people from snooping on other's subscriptions.
func (service *Subscription) LoadByID(userID primitive.ObjectID, subscriptionID primitive.ObjectID, result *model.Subscription) error {

	criteria := exp.Equal("_id", subscriptionID).
		AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Subscription.LoadByID", "Error loading Subscription", criteria)
	}

	return nil
}

// LoadByToken loads an individual subscription using a string version of the subscription ID
func (service *Subscription) LoadByToken(userID primitive.ObjectID, token string, result *model.Subscription) error {

	subscriptionID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "render.StepEditSubscription", "Error parsing subscriptionId", token)
	}

	return service.LoadByID(userID, subscriptionID, result)
}

/*******************************************
 * Subscription Methods
 *******************************************/

// CheckSubscriptions parses the RSS feed and adds/updates a new inboxItem for each item in it.
func (service *Subscription) CheckSubscription(subscription *model.Subscription) error {

	const location = "service.Subscription.PollSubscription"

	var body bytes.Buffer

	// Try to load the document from the remote site
	transaction := remote.Get(subscription.URL).Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return derp.Wrap(err, location, "Error fetching URL")
	}

	// Try to parse the document as an HTML document
	// TODO: if we have a direct URL for an RSS feed, then we should just use it straight up.
	document, err := goquery.NewDocumentFromReader(&body)

	if err != nil {
		return derp.Wrap(err, location, "Error parsing HTML")
	}

	// Look through RSS links for a valid feed
	for _, node := range document.Find("link[type='application/rss+xml']").Nodes {

		url := nodeAttribute(node, "href")
		err := service.LoadRSSFeed(subscription, url)

		if err == nil {
			break
		}

		derp.Report(err)
	}

	return nil
}

func (service *Subscription) LoadRSSFeed(subscription *model.Subscription, url string) error {

	const location = "service.Subscription.LoadRSSFeed"

	// Try to load the feed from the URL
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(url)

	// TODO: Limit retries on failed subscription URLs

	if err != nil {
		return derp.Wrap(err, location, "Error Parsing Feed URL")
	}

	// Update each inboxItem in the RSS feed
	for _, item := range feed.Items {
		if err := service.updateInboxItem(subscription, item); err != nil {
			return derp.Wrap(err, location, "Error updating local inboxItem")
		}
	}

	// Mark the subscription as having been polled
	subscription.MarkPolled()

	// Try to save the updated subscription to the database
	if err := service.Save(subscription, "Polling Subscription"); err != nil {
		return derp.Wrap(err, location, "Error saving subscription")
	}

	return nil
}

// updateStream adds/updates an individual InboxItem based on an RSS item
func (service *Subscription) updateInboxItem(subscription *model.Subscription, rssItem *gofeed.Item) error {

	const location = "service.Subscription.updateStream"

	inboxItem := model.NewInboxItem()

	if err := service.inboxService.LoadByOriginURL(subscription.UserID, rssItem.Link, &inboxItem); err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading local inboxItem")
		}

		// Fall through means "not found" which means "make a new inboxItem"
		inboxItem = model.NewInboxItem()
		inboxItem.UserID = subscription.UserID
		inboxItem.Origin = service.rssOrigin(rssItem)
	}

	// Calculate the update date.  Prefer Updated, then Published, then Now
	updateDate := time.Now().Unix()

	if rssItem.PublishedParsed != nil {
		updateDate = rssItem.PublishedParsed.Unix()
	}

	if rssItem.UpdatedParsed != nil {
		updateDate = rssItem.UpdatedParsed.Unix()
	}

	// If inboxItem has been updated since previous save, then set new values and update
	if inboxItem.Origin.UpdateDate > updateDate {

		// Populate inboxItem header and content
		inboxItem.Label = rssItem.Title
		inboxItem.Summary = rssItem.Description
		inboxItem.Content = rssItem.Content
		inboxItem.PublishDate = rssItem.PublishedParsed.Unix()
		inboxItem.Origin.UpdateDate = updateDate
		inboxItem.InboxFolderID = subscription.InboxFolderID
		inboxItem.Author = service.rssAuthor(rssItem)
		inboxItem.ImageURL = service.rssImageURL(rssItem)

		// Try to save the new/updated inboxItem
		if err := service.inboxService.Save(&inboxItem, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "service.Subscription.Poll", "Error saving inboxItem")
		}
	}

	return nil
}

// rssOrigin returns a popluated OriginLink for an RSS item
func (service *Subscription) rssOrigin(item *gofeed.Item) model.OriginLink {

	result := model.NewOriginLink()

	if item == nil {
		return result
	}

	result.Source = "RSS"
	result.Label = item.Title
	result.URL = item.Link
	result.UpdateDate = time.Now().Unix()

	return result
}

// rssAuthor returns all information about the author of an RSS item
func (service *Subscription) rssAuthor(item *gofeed.Item) model.AuthorLink {

	result := model.NewAuthorLink()

	if item == nil {
		return result
	}

	if item.Author != nil {
		result.Name = item.Author.Name
		result.EmailAddress = item.Author.Email
		return result
	}

	if len(item.Authors) > 0 {
		result.Name = item.Authors[0].Name
		result.EmailAddress = item.Authors[0].Email
		return result
	}

	return result
}

// rssImageURL returns the URL of the first image in the item's enclosure list.
func (service *Subscription) rssImageURL(item *gofeed.Item) string {

	if item == nil {
		return ""
	}

	if item.Image != nil {
		return item.Image.URL
	}

	// Search for an image in the enclosures
	for _, enclosure := range item.Enclosures {
		if list.Slash(enclosure.Type).Head() == "image" {
			return enclosure.URL
		}
	}

	return ""
}

// nodeAttribute searches for a specific attribute in a node and returns its value
func nodeAttribute(node *html.Node, name string) string {

	if node == nil {
		return ""
	}

	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}

	return ""
}
