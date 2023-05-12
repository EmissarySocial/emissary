package service

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data/option"
	"github.com/benpate/derp"
	"github.com/benpate/exp"
	"github.com/gorilla/feeds"
)

// RSS service generates RSS feeds of the available streams in the database
type RSS struct {
	streamService *Stream
	host          string
}

// NewRSS returns a fully initialized RSS service
func NewRSS(streamService *Stream, host string) *RSS {
	return &RSS{
		streamService: streamService,
		host:          host,
	}
}

// Feed generates an RSS data feed based on the provided query criteria.  This feed
// has a lot of incomplete data at the top level, so we're expecting the handler
// that calls this to fill in the rest of the gaps before it passes the values back
// to the requester.
func (rss RSS) Feed(criteria ...exp.Expression) (*feeds.JSONFeed, error) {

	filter := exp.And(criteria...)

	streams, err := rss.streamService.List(filter, option.SortDesc("publishDate"))
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

	result := &feeds.JSONItem{
		Id:            stream.Permalink(),
		Url:           stream.Permalink(),
		ExternalUrl:   stream.Permalink(),
		Title:         stream.Label,
		Summary:       stream.Summary,
		Image:         stream.ImageURL,
		PublishedDate: &publishDate,
		ModifiedDate:  &modifiedDate,
	}

	if !stream.AttributedTo.IsEmpty() {
		author := stream.AttributedTo.First()
		result.Author = &feeds.JSONAuthor{
			Name:   author.Name,
			Url:    author.ProfileURL,
			Avatar: author.ImageURL,
		}
	}

	return result
}
