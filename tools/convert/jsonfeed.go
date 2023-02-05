package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/EmissarySocial/emissary/tools/iterators"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/html"
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
		Items: iterators.Map(iterator, model.NewStream, StreamToJsonFeed),
	}
}

func StreamToJsonFeed(stream model.Stream) jsonfeed.Item {

	return jsonfeed.Item{
		ID:            stream.Token,
		URL:           stream.Document.URL,
		Title:         stream.Document.Label,
		ContentHTML:   first.String(stream.Content.HTML, " "),
		Summary:       stream.Document.Summary,
		Image:         stream.Document.ImageURL,
		DatePublished: time.UnixMilli(stream.PublishDate),
		DateModified:  time.UnixMilli(stream.UpdateDate),
		Author: &jsonfeed.Author{
			Name:   stream.Document.Author.Name,
			URL:    stream.Document.Author.ProfileURL,
			Avatar: stream.Document.Author.ImageURL,
		},
		// TODO: Attachments for podcasts, etc.
	}
}

func JsonFeedToActivity(feed jsonfeed.Feed, item jsonfeed.Item) model.Message {

	message := model.NewMessage()

	message.Origin = model.OriginLink{
		Label:    feed.Title,
		URL:      feed.HomePageURL,
		ImageURL: feed.Icon,
	}

	message.Document = model.DocumentLink{
		URL:         item.URL,
		Label:       item.Title,
		Summary:     item.Summary,
		ImageURL:    item.Image,
		PublishDate: item.DatePublished.UnixMilli(),
		Author:      JsonFeedToAuthor(feed, item),
	}

	if item.ContentHTML != "" {
		message.ContentHTML = item.ContentHTML
	} else if item.ContentText != "" {
		message.ContentHTML = html.FromText(item.ContentText)
	}

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
