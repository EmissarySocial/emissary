package convert

import (
	"bytes"
	"mime"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/remote"
	"github.com/benpate/rosetta/first"
	htmlTools "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/dyatlov/go-htmlinfo/htmlinfo"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

// RSSToActivity populates an Activity object from a gofeed.Feed and gofeed.Item
func RSSToActivity(feed *gofeed.Feed, rssItem *gofeed.Item) model.Activity {

	activity := model.NewInboxActivity()

	activity.Origin = model.OriginLink{
		URL:   feed.FeedLink,
		Label: feed.Title,
	}

	if feed.Image != nil {
		activity.Origin.ImageURL = feed.Image.URL
	}

	activity.Document = model.DocumentLink{
		URL:         rssItem.Link,
		Label:       htmlTools.ToText(rssItem.Title),
		Summary:     htmlTools.ToText(rssItem.Description),
		ImageURL:    rssImageURL(rssItem),
		Author:      rssAuthor(feed, rssItem),
		PublishDate: rssDate(rssItem.PublishedParsed),
		UpdateDate:  time.Now().Unix(),
	}
	activity.ContentHTML = bluemonday.UGCPolicy().Sanitize(rssItem.Content)

	// If there are fields missing from the RSS feed, try to fill them in from the web page
	if !activity.Document.IsComplete() {
		rssToActivity_populate(&activity)
	}

	return activity
}

// rssToActivity_populate loads the original web page to try to fill in missing data
func rssToActivity_populate(activity *model.Activity) {

	const location = "convert.RSSToActivity.populate"

	var body bytes.Buffer

	// Try to load the URL from the RSS feed
	txn := remote.Get(activity.Document.URL).Response(&body, nil)
	if err := txn.Send(); err != nil {
		derp.Report(derp.Wrap(err, location, "Error fetching URL", activity))
		return
	}

	// Parse the response into an HTMLInfo object
	mimeType := txn.ResponseObject.Header.Get("Content-Type")
	mediaType, _, _ := mime.ParseMediaType(mimeType)
	info := htmlinfo.NewHTMLInfo()

	if err := info.Parse(&body, &activity.Origin.URL, &mediaType); err != nil {
		derp.Report(derp.Wrap(err, location, "Error parsing HTML", activity.Origin.URL))
		return
	}

	// Update the activity with data missing from the RSS feed
	activity.Document.Label = first.String(activity.Document.Label, info.Title)
	activity.Document.Summary = first.String(activity.Document.Summary, info.Description)

	if activity.Document.ImageURL == "" {
		if info.ImageSrcURL != "" {
			activity.Document.ImageURL = info.ImageSrcURL
		} else if len(info.OGInfo.Images) > 0 {
			activity.Document.ImageURL = info.OGInfo.Images[0].URL
		}
	}

	// TODO: MEDIUM: Maybe implement h-feed in here?
	// https://indieweb.org/h-feed
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
