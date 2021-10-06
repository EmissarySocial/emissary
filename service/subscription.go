package service

import (
	"fmt"
	"time"

	"github.com/benpate/data"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/benpate/ghost/content"
	"github.com/benpate/ghost/model"
	"github.com/benpate/list"
	"github.com/davecgh/go-spew/spew"
	"github.com/mmcdole/gofeed"
)

// Subscription manages all interactions with the Subscription collection
type Subscription struct {
	collection    data.Collection
	streamService *Stream
}

// NewSubscription returns a fully populated Subscription service.
func NewSubscription(collection data.Collection, streamService *Stream) *Subscription {

	result := Subscription{
		collection:    collection,
		streamService: streamService,
	}

	go result.start()

	return &result
}

// New creates a newly initialized Subscription that is ready to use
func (service *Subscription) New() *model.Subscription {
	return model.NewSubscription()
}

func (service *Subscription) start() {

	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		fmt.Println(".. Polling Subscriptions")
		it, err := service.ListPollable()

		if err != nil {
			derp.Report(derp.Wrap(err, "ghost.service.Subscription.Run", "Error listing pollable subscriptions"))
			continue
		}

		subscription := model.Subscription{}

		for it.Next(&subscription) {
			service.pollSubscription(&subscription)
			subscription = model.Subscription{}
		}
	}
}

func (service *Subscription) pollSubscription(sub *model.Subscription) {
	// TODO: Check if subscription is past its polling window

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(sub.URL)

	if err != nil {
		derp.Report(derp.Wrap(err, "ghost.service.Subscription.Poll", "Error Parsing Feed URL"))
		return
	}

	for _, item := range feed.Items {
		if err := service.updateStream(sub, item); err != nil {
			derp.Report(derp.Wrap(err, "ghost.service.Subscription.Poll", "Error updating local stream"))
		}
	}
}

func (service *Subscription) updateStream(sub *model.Subscription, item *gofeed.Item) error {

	var stream model.Stream

	err := service.streamService.LoadBySource(sub.ParentStreamID, item.Link, &stream)

	if err != nil {

		// Anything but a "not found" error is a real error
		if !derp.NotFound(err) {
			return derp.Wrap(err, "ghost.service.Subscription.Poll", "Error loading local stream")
		}

		// Fall through means "not found" which means "make a new stream"
		stream = service.streamService.New()
		stream.TemplateID = "rss-article"
		stream.ParentID = sub.ParentStreamID
		stream.SourceURL = item.Link
		stream.StateID = "unread"
	}

	updateDate := item.PublishedParsed.Unix()

	if item.UpdatedParsed != nil {
		updateDate = item.UpdatedParsed.Unix()
	}

	// If stream has been updated since previous save, then set new values
	if true || stream.SourceUpdated != updateDate {

		stream.Label = item.Title
		stream.Description = item.Description
		stream.Content = content.FromHTML(item.Content)
		stream.PublishDate = item.PublishedParsed.Unix()
		stream.SourceUpdated = updateDate

		if item.Author == nil {
			stream.AuthorName = ""
			// stream.AuthorEmail = ""
		} else {
			stream.AuthorName = item.Author.Name
			// stream.AuthorEmail = item.Author.Email
		}

		if item.Image != nil {
			stream.ThumbnailImage = item.Image.URL
		} else {
			stream.ThumbnailImage = ""

			// Search for an image in the enclosures
			for _, enclosure := range item.Enclosures {
				spew.Dump(enclosure)
				if list.Head(enclosure.Type, "/") == "image" {
					stream.ThumbnailImage = enclosure.URL
					break
				}
			}
		}

		if err := service.streamService.Save(&stream, "Imported from RSS feed"); err != nil {
			return derp.Wrap(err, "ghost.service.Subscription.Poll", "Error saving stream")
		}
	}

	return nil
}

// List returns an iterator containing all of the Subscriptions who match the provided criteria
func (service *Subscription) List(criteria exp.Expression, options ...option.Option) (data.Iterator, error) {
	return service.collection.List(criteria, options...)
}

// Load retrieves an Subscription from the database
func (service *Subscription) Load(criteria exp.Expression, result *model.Subscription) error {

	if err := service.collection.Load(criteria, result); err != nil {
		return derp.Wrap(err, "ghost.service.Subscription", "Error loading Subscription", criteria)
	}

	return nil
}

// Save adds/updates an Subscription in the database
func (service *Subscription) Save(subscription *model.Subscription, note string) error {

	if err := service.collection.Save(subscription, note); err != nil {
		return derp.Wrap(err, "ghost.service.Subscription", "Error saving Subscription", subscription, note)
	}

	return nil
}

// Delete removes an Subscription from the database (virtual delete)
func (service *Subscription) Delete(subscription *model.Subscription, note string) error {

	if err := service.collection.Delete(subscription, note); err != nil {
		return derp.Wrap(err, "ghost.service.Subscription", "Error deleting Subscription", subscription, note)
	}

	return nil
}

// QUERIES //////////////////////////////////////

func (service *Subscription) ListPollable() (data.Iterator, error) {

	pollDuration := time.Now().Add(-1 * time.Hour).Unix()

	criteria := exp.Equal("journal.deleteDate", 0).
		AndLessThan("lastPolled", pollDuration)

	return service.List(criteria, option.SortAsc("lastPolled"))
}
