package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/hannibal/vocab"
	htmlTools "github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/list"
	"github.com/benpate/rosetta/mapof"
	"github.com/microcosm-cc/bluemonday"
	"github.com/mmcdole/gofeed"
)

// RSSToActivity populates an Activity object from a gofeed.Feed and gofeed.Item
func RSSToActivity(feed *gofeed.Feed, rssItem *gofeed.Item) mapof.Any {

	result := mapof.Any{
		vocab.PropertyID:        rssItem.Link,
		vocab.PropertyName:      htmlTools.ToText(rssItem.Title),
		vocab.PropertyPublished: rssDate(rssItem.PublishedParsed),
	}

	if summary := rssSummary(rssItem); summary != "" {
		result[vocab.PropertySummary] = summary
	}

	if imageURL := rssImageURL(feed, rssItem); imageURL != "" {
		result[vocab.PropertyImage] = imageURL
	}

	if contentHTML := rssContent(rssItem.Content); contentHTML != "" {
		result[vocab.PropertyContent] = contentHTML
	}

	if attributedTo := rssAuthor(feed, rssItem); attributedTo.NotEmpty() {
		result[vocab.PropertyAttributedTo] = attributedTo.GetJSONLD()
	}

	return result
}

func rssSummary(rssItem *gofeed.Item) string {
	return htmlTools.ToText(rssItem.Description)
}

func rssContent(value string) string {
	return bluemonday.UGCPolicy().Sanitize(value)
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
func rssImageURL(rssFeed *gofeed.Feed, rssItem *gofeed.Item) string {

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

	// Search for media extensions (YouTube uses this)
	if media, ok := rssItem.Extensions["media"]; ok {
		if group, ok := media["group"]; ok {
			for _, extension := range group {
				if thumbnails, ok := extension.Children["thumbnail"]; ok {
					for _, item := range thumbnails {
						if url := item.Attrs["url"]; url != "" {
							return url
						}
					}
				}
			}
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
