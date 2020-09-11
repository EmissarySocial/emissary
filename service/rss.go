package service

import (
	"time"

	"github.com/benpate/data/expression"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/gorilla/feeds"
)

// RSS service generates RSS feeds of the available streams in the database
type RSS struct {
	factory *Factory
}

// Feed generates an RSS data feed based on the provided query criteria.  This feed
// has a lot of incomplete data at the top level, so we're expecting the handler
// that calls this to fill in the rest of the gaps before it passes the values back
// to the requester.
func (rss RSS) Feed(criteria ...expression.Expression) (*feeds.JSONFeed, *derp.Error) {

	streamService := rss.factory.Stream()

	filter := expression.And(criteria...)

	streams, err := streamService.List(filter, option.SortDesc("publishDate"))
	stream := model.NewStream()

	if err != nil {
		return nil, derp.Wrap(err, "service.rss.Feed", "Error loading streams")
	}

	result := feeds.JSONFeed{
		Items: []*feeds.JSONItem{},
	}

	for streams.Next(&stream) {
		result.Items = append(result.Items, rss.Item(stream))
	}

	return &result, nil
}

// Item converts a single model.Stream into a feeds.JSONItem
func (rss RSS) Item(stream model.Stream) *feeds.JSONItem {

	publishDate := time.Unix(stream.PublishDate, 0)
	modifiedDate := time.Unix(stream.Journal.UpdateDate, 0)

	return &feeds.JSONItem{
		Id:            "",
		Url:           stream.URL,
		ExternalUrl:   stream.SourceURL,
		Title:         stream.Label,
		Summary:       stream.Description,
		Image:         stream.ThumbnailImage,
		PublishedDate: &publishDate,
		ModifiedDate:  &modifiedDate,

		Author: &feeds.JSONAuthor{
			Name: stream.AuthorName,
			Url:  stream.AuthorURL,
		},
	}
}
