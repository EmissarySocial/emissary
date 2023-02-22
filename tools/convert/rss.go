package convert

import (
	"bytes"
	"mime"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	htmlTools "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/dyatlov/go-htmlinfo/htmlinfo"
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

	message.Document = model.DocumentLink{
		URL:         rssItem.Link,
		Label:       htmlTools.ToText(rssItem.Title),
		Summary:     rssSummary(rssItem),
		ImageURL:    rssImageURL(rssItem),
		Author:      rssAuthor(feed, rssItem),
		PublishDate: rssDate(rssItem.PublishedParsed),
		UpdateDate:  time.Now().Unix(),
	}
	message.ContentHTML = bluemonday.UGCPolicy().Sanitize(rssItem.Content)

	// If there are fields missing from the RSS feed, try to fill them in from the web page
	if !message.Document.IsComplete() {
		rssToActivity_populate(&message)
	}

	return message
}

// rssToActivity_populate loads the original web page to try to fill in missing data
func rssToActivity_populate(message *model.Message) {

	const location = "convert.RSSToActivity.populate"

	var body bytes.Buffer

	// Try to load the URL from the RSS feed
	txn := remote.Get(message.Document.URL).Response(&body, nil)
	if err := txn.Send(); err != nil {
		derp.Report(derp.Wrap(err, location, "Error fetching URL", message))
		return
	}

	// Parse the response into an HTMLInfo object
	mimeType := txn.ResponseObject.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(mimeType)
	info := htmlinfo.NewHTMLInfo()

	if err := info.Parse(&body, &message.Origin.URL, &mediaType); err != nil {
		derp.Report(derp.Wrap(err, location, "Error parsing HTML", message.Origin.URL))
		return
	}
	// Update the message with data missing from the RSS feed
	// message.Document.Label = first.String(message.Document.Label, info.Title)
	// message.Document.Summary = first.String(message.Document.Summary, info.Description)

	if message.Document.ImageURL == "" {
		if info.ImageSrcURL != "" {
			message.Document.ImageURL = info.ImageSrcURL
		} else if len(info.OGInfo.Images) > 0 {
			message.Document.ImageURL = info.OGInfo.Images[0].URL
		}
	}

	// TODO: MEDIUM: Maybe implement h-feed in here?
	// https://indieweb.org/h-feed
}

func rssSummary(rssItem *gofeed.Item) string {

	if rssItem.Description != "" {
		htmlTools.ToText(rssItem.Description)
	}

	return ""
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
