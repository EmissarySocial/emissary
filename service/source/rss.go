package source

import (
	"github.com/benpate/derp"
	"github.com/benpate/ghost/model"
	"github.com/mmcdole/gofeed"
	"github.com/qri-io/jsonschema"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RSS is an adapter that downloads an RSS feed and converts it into a slice of model.Stream objects
type RSS struct {
	SourceID primitive.ObjectID
	URL      string
}

// Init populates this RSS feed with the configuraion data, and returns an error if the configuration data is invalid
func (rss *RSS) Init(sourceID primitive.ObjectID, config model.SourceConfig) error {

	rss.SourceID = sourceID

	if url, ok := config["url"]; ok {
		rss.URL = url
	} else {
		return derp.New(500, "service.rss.NewRSS", "Invalid URL parameter", config)
	}

	return nil
}

// JSONSchema returns a JSON-Schema object that can validate the configuration data required for this adapter
func (rss RSS) JSONSchema() jsonschema.Schema {
	return jsonschema.Schema{}
}

// JSONForm returns a JSON-Form object that can collect the configuration data required for this adapter
func (rss RSS) JSONForm() string {
	return ""
}

// Poll checks the remote data source and returnsa slice of model.Stream objects
func (rss RSS) Poll() ([]model.Stream, error) {

	var result []model.Stream

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rss.URL)

	if err != nil {
		return result, derp.New(500, "source.rss.GetStreams", "Error retrieving RSS feed", rss, err)
	}

	// Allocate memory for the slice and map all stream records into it.
	result = make([]model.Stream, len(feed.Items))

	for index, rssItem := range feed.Items {
		result[index] = rss.makeStream(rssItem)
	}

	return result, nil
}

func (rss RSS) Webhook(data map[string]interface{}) (model.Stream, error) {
	return model.Stream{}, nil
}

// makeStream maps data from a single RSS feed item into a model.Stream object.
func (rss RSS) makeStream(rssItem *gofeed.Item) model.Stream {

	stream := model.NewStream()
	stream.Title = rssItem.Title
	stream.AuthorName = rssItem.Author.Name
	stream.AuthorURL = rssItem.Author.Email
	stream.Tags = append(stream.Tags, rssItem.Categories...)
	stream.Summary = rssItem.Description
	stream.SourceID = rss.SourceID
	stream.SourceURL = rssItem.Link

	if rssItem.Image != nil {
		stream.Image = rssItem.Image.URL
	}

	if rssItem.PublishedParsed != nil {
		stream.PublishDate = rssItem.PublishedParsed.Unix()
	}

	if rssItem.UpdatedParsed != nil {
		stream.UpdateDate = rssItem.UpdatedParsed.Unix()
	}

	return stream
}
