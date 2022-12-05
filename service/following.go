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

// Following manages all interactions with the Following collection
type Following struct {
	collection   data.Collection
	userService  *User
	inboxService *Inbox
	queue        *queue.Queue
	closed       chan bool
}

// NewFollowing returns a fully populated Following service.
func NewFollowing(collection data.Collection, userService *User, inboxService *Inbox, queue *queue.Queue) Following {

	service := Following{
		collection:   collection,
		userService:  userService,
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
func (service *Following) Refresh(collection data.Collection) {
	service.collection = collection
}

// Close stops the following service watcher
func (service *Following) Close() {
	close(service.closed)
}

// Start begins the background scheduler that checks each following
// according to its own polling frequency
// TODO: HIGH: Need to make this configurable on a per-physical-server basis so that
// clusters can work together without hammering the Following collection.
func (service *Following) Start() {

	const location = "service.Following.Start"

	rand.Seed(time.Now().UnixNano())

	// query the database every minute, looking for following that should be loaded from the web.
	for {

		// Poll randomly between 1 and 5 minutes
		// time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
		time.Sleep(time.Duration(10 * time.Second))

		// If (for some reason) the service collection is still nil, then
		// wait this one out.
		if service.collection == nil {
			continue
		}

		// Get a list of all following that can be polled
		it, err := service.ListPollable()

		if err != nil {
			derp.Report(derp.Wrap(err, location, "Error listing pollable following"))
			continue
		}

		following := model.NewFollowing()

		for it.Next(&following) {
			select {

			// If we're done, we're done.
			case <-service.closed:
				return

			default:

				// Poll each following for new items.
				derp.Report(service.PollFollowing(following))
				derp.Report(service.PurgeFollowing(following))
			}

			following = model.NewFollowing()
		}

		// Poll randomly between 1 and 5 minutes
		time.Sleep(time.Duration(rand.Intn(5)+1) * time.Minute)
	}
}

/*******************************************
 * Common Data Methods
 *******************************************/

// New creates a newly initialized Following that is ready to use
func (service *Following) New() model.Following {
	return model.NewFollowing()
}

// Query returns an iterator containing all of the Following who match the provided criteria
func (service *Following) Query(criteria exp.Expression, options ...option.Option) ([]model.Activity, error) {
	result := make([]model.Activity, 0)
	err := service.collection.Query(&result, notDeleted(criteria), options...)
	return result, err
}

// List returns an iterator containing all of the Following who match the provided criteria
func (service *Following) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(notDeleted(criteria), options...)
}

// Load retrieves an Following from the database
func (service *Following) Load(criteria exp.Expression, result *model.Following) error {

	if err := service.collection.Load(notDeleted(criteria), result); err != nil {
		return derp.Wrap(err, "service.Following", "Error loading Following", criteria)
	}

	return nil
}

// Save adds/updates an Following in the database
func (service *Following) Save(following *model.Following, note string) error {

	// RULE: Reset status and error counts when saving
	following.ErrorCount = 0
	following.StatusMessage = ""
	following.Status = model.FollowingStatusNew

	// Clean the value before saving
	if err := service.Schema().Clean(following); err != nil {
		return derp.Wrap(err, "service.Following.Save", "Error cleaning Following", following)
	}

	// Save the following to the database
	if err := service.collection.Save(following, note); err != nil {
		return derp.Wrap(err, "service.Following", "Error saving Following", following, note)
	}

	// Recalculate the follower count for this user
	go service.userService.CalcFollowingCount(following.UserID)

	// Poll external services (if necessary)
	go service.PollFollowing(*following)

	// Win!
	return nil
}

// Delete removes an Following from the database (virtual delete)
func (service *Following) Delete(following *model.Following, note string) error {

	if err := service.collection.Delete(following, note); err != nil {
		return derp.Wrap(err, "service.Following", "Error deleting Following", following, note)
	}

	return nil
}

/*******************************************
 * Model Service Methods
 *******************************************/

// ObjectType returns the type of object that this service manages
func (service *Following) ObjectType() string {
	return "Following"
}

// New returns a fully initialized model.Stream as a data.Object.
func (service *Following) ObjectNew() data.Object {
	result := model.NewFollowing()
	return &result
}

func (service *Following) ObjectID(object data.Object) primitive.ObjectID {

	if following, ok := object.(*model.Following); ok {
		return following.FollowingID
	}

	return primitive.NilObjectID
}

func (service *Following) ObjectQuery(result any, criteria exp.Expression, options ...option.Option) error {
	return service.collection.Query(result, criteria, options...)
}

func (service *Following) ObjectList(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.List(criteria, options...)
}

func (service *Following) ObjectLoad(criteria exp.Expression) (data.Object, error) {
	result := model.NewFollowing()
	err := service.Load(criteria, &result)
	return &result, err
}

func (service *Following) ObjectSave(object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Save(following, note)
	}
	return derp.NewInternalError("service.Following", "ObjectSave", "Invalid object type", object)
}

func (service *Following) ObjectDelete(object data.Object, note string) error {
	if following, ok := object.(*model.Following); ok {
		return service.Delete(following, note)
	}
	return derp.NewInternalError("service.Following", "ObjectDelete", "Invalid object type", object)
}

func (service *Following) ObjectUserCan(object data.Object, authorization model.Authorization, action string) error {
	return derp.NewUnauthorizedError("service.Following", "Not Authorized")
}

func (service *Following) Schema() schema.Schema {
	return schema.New(model.FollowingSchema())
}

/*******************************************
 * Custom Queries
 *******************************************/

// QueryByUserID returns a slice of all following for a given user
func (service *Following) QueryByUserID(userID primitive.ObjectID) ([]model.FollowingSummary, error) {
	result := make([]model.FollowingSummary, 0)
	criteria := exp.Equal("userId", userID)
	err := service.collection.Query(&result, notDeleted(criteria))
	return result, err
}

// ListPollable returns an iterator of all following that are ready to be polled
func (service *Following) ListPollable() (data.Iterator, error) {
	criteria := exp.LessThan("nextPoll", time.Now().Unix())
	return service.List(criteria, option.SortAsc("lastPolled"))
}

// ListByUserID returns an iterator of all following for a given user
func (service *Following) ListByUserID(userID primitive.ObjectID) (data.Iterator, error) {
	criteria := exp.Equal("userId", userID)
	return service.List(criteria, option.SortAsc("lastPolled"))
}

// LoadByID retrieves an Following from the database.  UserID is required to prevent
// people from snooping on other's following.
func (service *Following) LoadByID(userID primitive.ObjectID, followingID primitive.ObjectID, result *model.Following) error {

	criteria := exp.Equal("_id", followingID).
		AndEqual("userId", userID)

	if err := service.Load(criteria, result); err != nil {
		return derp.Wrap(err, "service.Following.LoadByID", "Error loading Following", criteria)
	}

	return nil
}

// LoadByToken loads an individual following using a string version of the following ID
func (service *Following) LoadByToken(userID primitive.ObjectID, token string, result *model.Following) error {

	if token == "new" {
		*result = model.NewFollowing()
		result.UserID = userID
		return nil
	}

	followingID, err := primitive.ObjectIDFromHex(token)

	if err != nil {
		return derp.Wrap(err, "service.Following.LoadByToken", "Error parsing followingId", token)
	}

	return service.LoadByID(userID, followingID, result)
}

/*******************************************
 * Following Methods
 *******************************************/

func (service *Following) PurgeFollowing(following model.Following) error {

	const location = "service.Following.PurgeFollowing"

	// Check each following for expired items.
	items, err := service.inboxService.QueryPurgeable(&following)

	// If there was an error querying for purgeable items, log it and exit.
	if err != nil {
		return derp.Wrap(err, location, "Error querying purgeable items", following)
	}

	// Purge each item that has expired
	for _, item := range items {
		if err := service.inboxService.Delete(&item, "Purged"); err != nil {
			return derp.Wrap(err, location, "Error purging item", item)
		}
	}

	return nil
}

// PollFollowing tries to import an RSS feed and adds/updates activitys for each item in it.
func (service *Following) PollFollowing(following model.Following) error {

	const location = "service.Following.PollFollowing"

	// Update the following status
	if err := service.SetStatus(&following, model.FollowingStatusLoading, ""); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	// Find the RSS feed associated with this following
	rssFeed, err := service.GetRSSFeed(&following)

	if err != nil {
		// If we can't find the RSS feed URL, then mark the following as an error.
		if err := service.SetStatus(&following, model.FollowingStatusFailure, err.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating following status", following)
		}
		return nil // Error has been logged, so don't break the request.
	}

	// If we have a feed, then import all of the items from it.

	// Update all items in the feed.  If we have an error, then don't stop, just save it for later.
	var errorCollection error

	for _, item := range rssFeed.Items {
		if err := service.saveActivity(&following, rssFeed, item); err != nil {
			errorCollection = derp.Append(errorCollection, derp.Wrap(err, location, "Error updating local activity"))
		}
	}

	// If there were errors parsing the feed, then mark the following as an error.
	if errorCollection != nil {

		// Try to update the following status
		if err := service.SetStatus(&following, model.FollowingStatusFailure, errorCollection.Error()); err != nil {
			return derp.Wrap(err, location, "Error updating following status", following)
		}

		// There were errors, but they're noted in the following status, so THIS step is successful
		return nil
	}

	// If we're here, then we have successfully imported the RSS feed.
	// Mark the following as having been polled
	if err := service.SetStatus(&following, model.FollowingStatusSuccess, ""); err != nil {
		return derp.Wrap(err, location, "Error updating following status", following)
	}

	return nil
}

// SetStatus updates the status (and statusMessage) of a Following record.
func (service *Following) SetStatus(following *model.Following, status string, statusMessage string) error {

	// RULE: Default Poll Duration is 24 hours
	if following.PollDuration == 0 {
		following.PollDuration = 24
	}

	// RULE: Require that poll duration is at least 1 hour
	if following.PollDuration < 1 {
		following.PollDuration = 1
	}

	// Update properties of the Following
	following.Status = status
	following.StatusMessage = statusMessage

	// Recalculate the next poll time
	switch following.Status {
	case model.FollowingStatusSuccess:

		// On success, "LastPolled" is only updated when we're successful.  Reset other times.
		following.LastPolled = time.Now().Unix()
		following.NextPoll = following.LastPolled + int64(following.PollDuration*60)
		following.ErrorCount = 0

	case model.FollowingStatusFailure:

		// On failure, compute exponential backoff
		// Wait times are 1m, 2m, 4m, 8m, 16m, 32m, 64m, 128m, 256m
		// But do not change "LastPolled" because that is the last time we were successful
		errorBackoff := following.ErrorCount

		if errorBackoff > 8 {
			errorBackoff = 8
		}

		errorBackoff = 2 ^ errorBackoff

		following.NextPoll = time.Now().Add(time.Duration(errorBackoff) * time.Minute).Unix()
		following.ErrorCount++

	default:
		// On all other statuse, the error counters are not touched
		// because "New" and "Loading" are going to be overwritten very soon.
	}

	// Try to save the Following to the database
	if err := service.collection.Save(following, "Updating status to loading"); err != nil {
		return derp.Wrap(err, "service.Following", "Error updating following status", following)
	}

	// Success!!
	return nil
}

func (service *Following) GetRSSFeed(following *model.Following) (*gofeed.Feed, error) {

	const location = "service.Following.GetRSSFeed"

	var body bytes.Buffer

	// Try to load the URL from the following
	transaction := remote.Get(following.URL).Response(&body, nil)

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

			href := getRelativeURL(following.URL, nodeAttribute(link, "href"))
			if rssFeed, err := gofeed.NewParser().ParseURL(href); err == nil {
				return rssFeed, nil
			}
		}
	}

	// Fall through means that we couldn't find a valid feed ANYWHERE.
	return nil, derp.NewBadRequestError(location, "RSS Feed Not Found", following.URL)
}

// saveActivity adds/updates an individual Activity based on an RSS item
func (service *Following) saveActivity(following *model.Following, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	const location = "service.Following.saveActivity"

	activity := model.NewActivity()

	if err := service.inboxService.LoadByOriginURL(following.UserID, rssItem.Link, &activity); err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, location, "Error loading local activity")
		}

		// Fall through means "not found" which means "make a new activity"
		activity.OwnerID = following.UserID
		activity.Origin = following.Origin()
		activity.PublishDate = rssDate(rssItem.PublishedParsed)
		activity.FolderID = following.FolderID

		if updateDate := rssDate(rssItem.UpdatedParsed); updateDate > activity.PublishDate {
			activity.PublishDate = updateDate
		}
	}

	// If the RSS entry has been updated since the Activity was last touched, then refresh it.
	if rssDate(rssItem.PublishedParsed) >= activity.Journal.UpdateDate {

		populateActivity(&activity, following, rssFeed, rssItem)

		// Try to save the new/updated activity
		if err := service.inboxService.Save(&activity, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "service.Following.Poll", "Error saving activity")
		}
	}

	return nil
}

/*******************************************
 * Helper Functions
 *******************************************/

func populateActivity(activity *model.Activity, following *model.Following, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	// Populate activity from the rssItem
	activity.PublishDate = rssDate(rssItem.PublishedParsed)
	activity.Origin = following.Origin()
	activity.Actor = rssActor(rssFeed, rssItem)
	activity.Object = rssDocument(rssItem)

	// Fill in additional properties from the web page, if necessary
	if !activity.Object.IsComplete() {

		var body bytes.Buffer

		// Try to load the URL from the RSS feed
		txn := remote.Get(activity.Origin.URL).Response(&body, nil)
		if err := txn.Send(); err != nil {
			return derp.Wrap(err, "service.Following.populateActivity", "Error fetching URL", activity.Origin.URL)
		}

		// Parse the response into an HTMLInfo object
		contentType := txn.ResponseObject.Header.Get("Content-Type")
		info := htmlinfo.NewHTMLInfo()

		if err := info.Parse(&body, &activity.Origin.URL, &contentType); err != nil {
			return derp.Wrap(err, "service.Following.populateActivity", "Error parsing HTML", activity.Origin.URL)
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

// rssActor returns all information about the actor of an RSS item
func rssActor(rssFeed *gofeed.Feed, rssItem *gofeed.Item) model.PersonLink {

	if rssFeed == nil {
		return model.NewPersonLink()
	}

	result := model.PersonLink{
		Organization: htmlTools.ToText(rssFeed.Title),
		Name:         htmlTools.ToText(rssItem.Author.Name),
		ProfileURL:   rssFeed.Link,
	}

	if rssFeed.Image != nil {
		result.ImageURL = rssFeed.Image.URL
	}

	return result
}

func rssDocument(rssItem *gofeed.Item) model.DocumentLink {

	return model.DocumentLink{
		URL:         rssItem.Link,
		Label:       htmlTools.ToText(rssItem.Title),
		Summary:     htmlTools.ToText(rssItem.Description),
		ContentHTML: bluemonday.UGCPolicy().Sanitize(rssItem.Content),
		ImageURL:    rssImageURL(rssItem),
		PublishDate: rssDate(rssItem.PublishedParsed),
		UpdateDate:  time.Now().Unix(),
	}
}

// rssImageURL returns the URL of the first image in the item's enclosure list.
func rssImageURL(rssItem *gofeed.Item) string {

	if rssItem == nil {
		return ""
	}

	if rssItem.Image != nil {
		return rssItem.Image.URL
	}

	// Search for an image in the enclosures
	for _, enclosure := range rssItem.Enclosures {
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
