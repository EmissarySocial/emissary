package service

import (
	"bytes"
	"math/rand"
	"net/url"
	"strings"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/queue"
	"github.com/PuerkitoBio/goquery"
	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	htmlTools "github.com/benpate/rosetta/html"
	"github.com/dyatlov/go-htmlinfo/htmlinfo"

	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/schema"
	"github.com/microcosm-cc/bluemonday"
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
// TODO: HIGH: Need to make this configurable on a per-physical-server basis so that
// clusters can work together without hammering the Subscription collection.
func (service *Subscription) Start() {

	const location = "service.Subscription.Start"

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
			derp.Report(derp.Wrap(err, location, "Error listing pollable subscriptions"))
			continue
		}

		subscription := model.NewSubscription()

		for it.Next(&subscription) {
			select {

			// If we're done, we're done.
			case <-service.closed:
				return

			default:

				// Poll each subscription for new items.
				derp.Report(service.PollSubscription(subscription))
				derp.Report(service.PurgeSubscriptions(subscription))
			}

			subscription = model.NewSubscription()
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

// Query returns an iterator containing all of the Subscriptions who match the provided criteria
func (service *Subscription) Query(criteria exp.Expression, options ...option.Option) ([]model.Activity, error) {
	result := make([]model.Activity, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
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

	subscription.ErrorCount = 0
	subscription.StatusMessage = ""
	subscription.Status = model.SubscriptionStatusNew

	// TODO: HIGH: Use schema to clean the model object before saving

	if err := service.collection.Save(subscription, note); err != nil {
		return derp.Wrap(err, "service.Subscription", "Error saving Subscription", subscription, note)
	}

	go service.PollSubscription(*subscription)

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
 * Model Service Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Subscription) ObjectType() string {
	return "Subscription"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Subscription) ObjectNew() data.Object {
	result := model.NewSubscription()
	return &result
}

func (service *Subscription) ObjectID(object data.Object) primitive.ObjectID {

	if subscription, ok := object.(*model.Subscription); ok {
		return subscription.SubscriptionID
	}

	return primitive.NilObjectID
}

func (service *Subscription) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Subscription) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewSubscription()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Subscription) ObjectSave(object data.Object, note string) error {
	if subscription, ok := object.(*model.Subscription); ok {
		return service.Save(subscription, note)
	}
	return derp.NewInternalError("service.Subscription", "ObjectSave", "Invalid object type", object)
}

func (service *Subscription) ObjectDelete(object data.Object, note string) error {
	if subscription, ok := object.(*model.Subscription); ok {
		return service.Delete(subscription, note)
	}
	return derp.NewInternalError("service.Subscription", "ObjectDelete", "Invalid object type", object)
}

func (service *Subscription) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Subscription", "Not Authorized")
}

func (service *Subscription) Schema() schema.Schema {
	return schema.New(model.SubscriptionSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

// QueryByUserID returns a slice of all subscriptions for a given user
func (service *Subscription) QueryByUserID(userID primitive.ObjectID) ([]model.SubscriptionSummary, error) {
	result := make([]model.SubscriptionSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection.Query(&result, notDeleted(criteria))
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

	if token == "new" {
		*result = model.NewSubscription()
		result.UserID = userID
		return nil
	}

	subscriptionID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Subscription.LoadByToken", "Error parsing subscriptionId", token)
	}

	return service.LoadByID(userID, subscriptionID, result)
}

/*******************************************
 * Subscription Methods
 *******************************************/

func (service *Subscription) PurgeSubscriptions(subscription model.Subscription) error {

	const location = "service.Subscription.PurgeSubscriptions"

	// Check each subscription for expired items.
	items, err := service.inboxService.QueryPurgeable(&subscription)

	// If there was an error querying for purgeable items, log it and exit.
	if err != nil {
		return derp.Wrap(err, location, "Error querying purgeable items", subscription)
	}

	// Purge each item that has expired
	for _, item := range items {
		if err := service.inboxService.Delete(&item, "Purged"); err != nil {
			return derp.Wrap(err, location, "Error purging item", item)
		}
	}

	return nil
}

// PollSubscriptions tries to import an RSS feed and adds/updates activitys for each item in it.
func (service *Subscription) PollSubscription(subscription model.Subscription) error {

	const location = "service.Subscription.PollSubscription"

	// Update the subscription status
	if err := service.SetStatus(&subscription, model.SubscriptionStatusLoading, ""); err != nil {
		return derp.Wrap(err, location, "Error updating subscription status", subscription)
	}

	// Find the RSS feed associated with this subscription
	rssFeed, err := service.GetRSSFeed(&subscription)

	if err != nil {
		// If we can't find the RSS feed URL, then mark the subscription as an error.
		if err := service.SetStatus(&subscription, model.SubscriptionStatusFailure, err.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating subscription status", subscription)
		}
		return nil // Error has been logged, so don't break the request.
	}

	// If we have a feed, then import all of the items from it.

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, item := range rssFeed.Items {
		if err := service.saveActivity(&subscription, rssFeed, item); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error updating local activity"))
		}
	}

	// If there were errors parsing the feed, then mark the subscription as an error.
	if errorCollection != nil {

		// Try to update the subscription status
		if err := service.SetStatus(&subscription, model.SubscriptionStatusFailure, errorCollection.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating subscription status", subscription)
		}

		// There were errors, but they're noted in the subscription status, so THIS step is successful
		return nil
	}

	// If we're here, then we have successfully imported the RSS feed.
	// Mark the subscription as having been polled
	if err := service.SetStatus(&subscription, model.SubscriptionStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error updating subscription status", subscription)
	}

	return nil
}

// SetStatus updates the status (and statusMessage) of a Subscription
func (service *Subscription) SetStatus(subscription *model.Subscription, status string, statusMessage string) error {

	// RULE: Default Poll Duration is 24 hours
	if subscription.PollDuration == 0 {
		subscription.PollDuration = 24
	}

	// RULE: Require that poll duration is at least 1 hour
	if subscription.PollDuration < 1 {
		subscription.PollDuration = 1
	}

	// Update properties of the Subscription
	subscription.Status = status
	subscription.StatusMessage = statusMessage

	// Recalculate the next poll time
	switch subscription.Status {
	case model.SubscriptionStatusSuccess:

		// On success, "LastPolled" is only updated when we're successful.  Reset other times.
		subscription.LastPolled = time.Now().Unix()
		subscription.NextPoll = subscription.LastPolled + int64(subscription.PollDuration*60)
		subscription.ErrorCount = 0

	case model.SubscriptionStatusFailure:

		// On failure, compute exponential backoff
		// Wait times are 1m, 2m, 4m, 8m, 16m, 32m, 64m, 128m, 256m
		// But do not change "LastPolled" because that is the last time we were successful
		errorBackoff := subscription.ErrorCount

		if errorBackoff > 8 {
			errorBackoff = 8
		}

		errorBackoff = 2 ^ errorBackoff

		subscription.NextPoll = time.Now().Add(time.Duration(errorBackoff) * time.Minute).Unix()
		subscription.ErrorCount++

	default:
		// On all other statuse, the error counters are not touched
		// because "New" and "Loading" are going to be overwritten very soon.
	}

	// Try to save the Subscription to the database
	if err := service.collection.Save(subscription, "Updating status to loading"); err != nil {
		return derp.Wrap(err, "service.Subscription", "Error updating subscription status", subscription)
	}

	// Success!!
	return nil
}

func (service *Subscription) GetRSSFeed(subscription *model.Subscription) (*gofeed.Feed, error) {

	const location = "service.Subscription.GetRSSFeed"

	var body bytes.Buffer

	// Try to load the URL from the subscription
	transaction := remote.Get(subscription.URL).Response(&body, nil)

	if err := transaction.Send(); err != nil {
		return nil, derp.Wrap(err, location, "Error fetching URL")
	}

	// If it is a valid RSS feed, then we have won.  NOTE: We're not checking the
	// document mime type becuase that's confusing and not reliable.
	if rssFeed, err := gofeed.NewParser().ParseString(body.String()); err == nil {
		return rssFeed, nil
	}

	// Fall through means that it was not a valid feed.  Maybe it's a HTML document that
	// LINKS to a feed.  Let's try that...

	// Try to parse the document as an HTML document
	htmlDocument, err := goquery.NewDocumentFromReader(&body)

	if err != nil {
		return nil, derp.Report(derp.Wrap(err, location, "Error parsing HTML document"))
	}

	links := htmlDocument.Find("link").Nodes

	// Look through RSS links for a valid feed
	for _, link := range links {

		// Only follow links that say they are some kind of "RSS" feed
		switch nodeAttribute(link, "type") {
		case "application/rss+xml", "application/atom+xml", "application/json+feed":

			href := getRelativeURL(subscription.URL, nodeAttribute(link, "href"))
			if rssFeed, err := gofeed.NewParser().ParseURL(href); err == nil {
				return rssFeed, nil
			}
		}
	}

	// Fall through means that we couldn't find a valid feed ANYWHERE.
	return nil, derp.NewBadRequestError(location, "RSS Feed Not Found", subscription.URL)
}

// saveActivity adds/updates an individual Activity based on an RSS item
func (service *Subscription) saveActivity(subscription *model.Subscription, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	const location = "service.Subscription.saveActivity"

	activity := model.NewActivity()

	if err := service.inboxService.LoadByOriginURL(subscription.UserID, rssItem.Link, &activity); err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading local activity")
		}

		// Fall through means "not found" which means "make a new activity"
		activity = model.NewActivity()
		activity.OwnerID = subscription.UserID
		activity.Origin = subscription.Origin()

		activity.PublishDate = rssDate(rssItem.PublishedParsed)

		if updateDate := rssDate(rssItem.UpdatedParsed); updateDate > activity.PublishDate {
			activity.PublishDate = updateDate
		}
	}

	// If the RSS entry has been updated since the Activity was last touched, then refresh it.
	if rssDate(rssItem.PublishedParsed) >= activity.Journal.UpdateDate {

		populateActivity(&activity, subscription, rssFeed, rssItem)

		// Try to save the new/updated activity
		if err := service.inboxService.Save(&activity, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "service.Subscription.Poll", "Error saving activity")
		}
	}

	return nil
}

/*******************************************
 * Helper Functions
 *******************************************/

func populateActivity(activity *model.Activity, subscription *model.Subscription, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	// Populate activity from the rssItem
	activity.PublishDate = rssDate(rssItem.PublishedParsed)
	activity.Origin = subscription.Origin()
	activity.FolderID = subscription.FolderID
	activity.Actor = rssAuthor(rssFeed)
	activity.Object = rssDocument(rssItem)

	// Fill in additional properties from the web page, if necessary
	if !activity.Object.IsComplete() {

		var body bytes.Buffer

		// Try to load the URL from the RSS feed
		txn := remote.Get(activity.Origin.URL).Response(&body, nil)
		if err := txn.Send(); err != nil {
			return derp.Wrap(err, "service.Subscription.populateActivity", "Error fetching URL", activity.Origin.URL)
		}

		// Parse the response into an HTMLInfo object
		contentType := txn.ResponseObject.Header.Get("Content-Type")
		info := htmlinfo.NewHTMLInfo()

		if err := info.Parse(&body, &activity.Origin.URL, &contentType); err != nil {
			return derp.Wrap(err, "service.Subscription.populateActivity", "Error parsing HTML", activity.Origin.URL)
		}

		// Update the activity with data missing from the RSS feed
		activity.Object.Label = first.String(activity.Object.Label, info.Title)
		activity.Object.Summary = first.String(activity.Object.Summary, info.Description)

		if activity.Object.ImageURL == "" {
			if info.ImageSrcURL != "" {
				activity.Object.ImageURL = info.ImageSrcURL
			} else if len(info.OGInfo.Images) > 0 {
				activity.Object.ImageURL = info.OGInfo.Images[0].URL
			}
		}

		// TODO: MEDIUM: Maybe implement h-feed in here?
		// https://indieweb.org/h-feed
	}

	return nil
}

// rssAuthor returns all information about the author of an RSS item
func rssAuthor(feed *gofeed.Feed) model.PersonLink {

	if feed == nil {
		return model.NewPersonLink()
	}

	result := model.PersonLink{
		Name:       htmlTools.ToText(feed.Title),
		ProfileURL: feed.Link,
	}

	if feed.Image != nil {
		result.ImageURL = feed.Image.URL
	}

	return result
}

func rssDocument(item *gofeed.Item) model.DocumentLink {

	return model.DocumentLink{
		URL:         item.Link,
		Label:       htmlTools.ToText(item.Title),
		Summary:     htmlTools.ToText(item.Description),
		ContentHTML: bluemonday.UGCPolicy().Sanitize(item.Content),
		ImageURL:    rssImageURL(item),
		PublishDate: rssDate(item.PublishedParsed),
		UpdateDate:  time.Now().Unix(),
	}
}

// rssImageURL returns the URL of the first image in the item's enclosure list.
func rssImageURL(item *gofeed.Item) string {

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

func rssDate(date *time.Time) int64 {

	if date == nil {
		return 0
	}

	return date.Unix()
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

func getRelativeURL(baseURL string, relativeURL string) string {

	// If the relative URL is already absolute, then just return it
	if strings.HasPrefix(relativeURL, "http://") || strings.HasPrefix(relativeURL, "https://") {
		return relativeURL
	}

	// If the relative URL is a root-relative URL, then assume HTTPS (it's 2022, for crying out loud)
	if strings.HasPrefix(relativeURL, "//") {
		return "https:" + relativeURL
	}

	if result, err := url.JoinPath(baseURL, relativeURL); err == nil {
		return result
	}

	return relativeURL
}
