package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	htmlTools "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

// RSSToActivity populates an Activity object from a gofeed.Feed and gofeed.Item
func RSSToActivity(feed *gofeed.Feed, rssItem *gofeed.Item) model.Message {

	message := model.NewMessage()

	message.Origin = model.OriginLink{
		URL:   feed.FeedLink,
		Label: feed.Title,
	}

	if feed.Image != nil {
		message.Origin.ImageURL = feed.Image.URL
	}

	message.URL = rssItem.Link
	message.Label = htmlTools.ToText(rssItem.Title)
	message.Summary = rssSummary(rssItem)
	message.ImageURL = rssImageURL(rssItem)
	message.ContentHTML = bluemonday.UGCPolicy().Sanitize(rssItem.Content)
	message.PublishDate = rssDate(rssItem.PublishedParsed)
	message.SetAttributedTo(rssAuthor(feed, rssItem))

	return message
}

func rssSummary(rssItem *gofeed.Item) string {
	return rssItem.Description
}

// rssAuthor returns all information about the actor of an RSS item
func rssAuthor(feed *gofeed.Feed, rssItem *gofeed.Item) model.PersonLink {

	if feed == nil {
		return model.NewPersonLink()
	}

	if rssItem == nil {
		return model.NewPersonLink()
	}

	result := model.PersonLink{
		ProfileURL: feed.Link,
	}

	if rssItem.Author != nil {
		result.Name = htmlTools.ToText(rssItem.Author.Name)
		result.EmailAddress = rssItem.Author.Email
	} else {
		result.Name = htmlTools.ToText(feed.Title)
	}

	// Look in the feed.Image object for an author image
	if feed.Image != nil {
		result.ImageURL = feed.Image.URL
	}

	// If we STILL don't have an author image, then try the "webfeeds" extension...
	if result.ImageURL == "" {
		if webfeeds, ok := feed.Extensions["webfeeds"]; ok {
			if icon, ok := webfeeds["icon"]; ok {
				for _, element := range icon {
					if element.Name == "icon" {
						result.ImageURL = element.Value
						break
					}
				}
			}
		}
	}

	return result
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
		if list.Slash(enclosure.Type).First() == "image" {
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
