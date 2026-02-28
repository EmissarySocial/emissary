package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/rosetta/first"
	"github.com/benpate/rosetta/iterator"
	"github.com/benpate/rosetta/slice"
	"github.com/kr/jsonfeed"
)

func IteratorToJSonFeed(url string, title string, description string, it data.Iterator) jsonfeed.Feed {

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
		Items: slice.Map(iterator.Slice(it, model.NewStream), StreamToJsonFeed),
	}
}

func StreamToJsonFeed(stream model.Stream) jsonfeed.Item {

	result := jsonfeed.Item{
		ID:            stream.Token,
		URL:           stream.URL,
		Title:         stream.Label,
		ContentHTML:   first.String(stream.Content.HTML, " "),
		Summary:       stream.Summary,
		Image:         stream.IconURL,
		DatePublished: time.Unix(stream.PublishDate, 0),
		DateModified:  time.UnixMilli(stream.UpdateDate),
	}

	// Attach author if available
	if stream.AttributedTo.NotEmpty() {
		result.Author = &jsonfeed.Author{
			Name:   stream.AttributedTo.Name,
			URL:    stream.AttributedTo.ProfileURL,
			Avatar: stream.AttributedTo.IconURL,
		}
	}

	// TODO: LOW: Attachments for podcasts, etc.

	return result
}
