package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/html"
	"github.com/benpate/rosetta/slice"
	"github.com/kr/jsonfeed"
)

func IteratorToJSonFeed(url string, title string, description string, iterator data.Iterator) jsonfeed.Feed {

	return jsonfeed.Feed{
		Version:     "https://jsonfeed.org/version/1.1",
		Title:       title,
		Description: description,
		HomePageURL: url,
		FeedURL:     url + "/feed?format=json",
		Hubs: []jsonfeed.Hub{{
			Type: "WebSub",
			URL:  url + "/websub",
		}},
		Items: slice.Map(data.Slice(iterator, model.NewStream), StreamToJsonFeed),
	}
}

func StreamToJsonFeed(stream model.Stream) jsonfeed.Item {

	result := jsonfeed.Item{
		ID:            stream.Token,
		URL:           stream.Document.URL,
		Title:         stream.Document.Label,
		ContentHTML:   first.String(stream.Content.HTML, " "),
		Summary:       stream.Document.Summary,
		Image:         stream.Document.ImageURL,
		DatePublished: time.UnixMilli(stream.PublishDate),
		DateModified:  time.UnixMilli(stream.UpdateDate),
	}

	// Attach author if available
	if !stream.Document.AttributedTo.IsEmpty() {
		author := stream.Document.AttributedTo.First()
		result.Author = &jsonfeed.Author{
			Name:   author.Name,
			URL:    author.ProfileURL,
			Avatar: author.ImageURL,
		}
	}

	// TODO: LOW: Attachments for podcasts, etc.

	return result
}

func JsonFeedToActivity(feed jsonfeed.Feed, item jsonfeed.Item) model.Message {

	message := model.NewMessage()

	message.Origin = model.OriginLink{
		Label:    feed.Title,
		URL:      feed.HomePageURL,
		ImageURL: feed.Icon,
	}

	message.Document = model.DocumentLink{
		URL:      item.URL,
		Label:    item.Title,
		Summary:  item.Summary,
		ImageURL: item.Image,
	}

	message.SetAttributedTo(JsonFeedToAuthor(feed, item))
	message.PublishDate = item.DatePublished.UnixMilli()
	message.ContentHTML = JsonFeedToContentHTML(item)

	return message
}

func JsonFeedToAuthor(feed jsonfeed.Feed, item jsonfeed.Item) model.PersonLink {

	result := model.NewPersonLink()

	if feed.Author != nil {
		result.Name = feed.Author.Name
		result.ProfileURL = feed.Author.URL
		result.ImageURL = feed.Author.Avatar
	}

	if item.Author != nil {
		result.Name = first.String(item.Author.Name, result.Name)
		result.ProfileURL = first.String(item.Author.URL, result.ProfileURL)
		result.ImageURL = first.String(item.Author.Avatar, result.ImageURL)
	}

	return result
}

func JsonFeedToContentHTML(item jsonfeed.Item) string {

	var result string

	if item.ContentHTML != "" {
		result = item.ContentHTML
	} else if item.ContentText != "" {
		result = html.FromText(item.ContentText)
	}

	return SanitizeHTML(result)
}
