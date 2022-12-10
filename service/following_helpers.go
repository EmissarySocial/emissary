package service

import (
	"bytes"
	"net/url"
	"strings"
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
	"golang.org/x/net/html"
)

/*******************************************
 * Helper Functions
 *******************************************/

func populateActivity(activity *model.Activity, following *model.Following, rssFeed *gofeed.Feed, rssItem *gofeed.Item) error {

	// Populate activity from the rssItem
	activity.PublishDate = rssDate(rssItem.PublishedParsed)
	activity.Origin = following.Origin()
	activity.Document = rssDocument(rssFeed, rssItem)
	activity.ContentHTML = bluemonday.UGCPolicy().Sanitize(rssItem.Content)

	// Fill in additional properties from the web page, if necessary
	if !activity.Document.IsComplete() {

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

	return nil
}

func rssDocument(rssFeed *gofeed.Feed, rssItem *gofeed.Item) model.DocumentLink {

	return model.DocumentLink{
		URL:         rssItem.Link,
		Label:       htmlTools.ToText(rssItem.Title),
		Summary:     htmlTools.ToText(rssItem.Description),
		ImageURL:    rssImageURL(rssItem),
		Author:      rssAuthor(rssFeed, rssItem),
		PublishDate: rssDate(rssItem.PublishedParsed),
		UpdateDate:  time.Now().Unix(),
	}
}

// rssAuthor returns all information about the actor of an RSS item
func rssAuthor(rssFeed *gofeed.Feed, rssItem *gofeed.Item) model.PersonLink {

	if rssFeed == nil {
		return model.NewPersonLink()
	}

	result := model.PersonLink{
		Name:       htmlTools.ToText(rssItem.Author.Name),
		ProfileURL: rssFeed.Link,
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
