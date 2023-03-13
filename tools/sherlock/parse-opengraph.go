package sherlock

import (
	"io"

	"github.com/benpate/derp"
	"github.com/dyatlov/go-opengraph/opengraph"
)

func parseOpenGraph(url string, reader io.Reader, data *Page) {

	ogInfo := opengraph.NewOpenGraph()

	if err := ogInfo.ProcessHTML(reader); err != nil {
		derp.Report(derp.Wrap(err, "urlmeta.loadOpenGraph", "Error parsing HTML", url))
		return
	}

	if data.Type == "" {
		data.Type = ogInfo.Type
	}

	if data.Title == "" {
		data.Title = ogInfo.Title
	}

	if data.Description == "" {
		data.Description = ogInfo.Description
	}

	if data.ProviderName == "" {
		data.ProviderName = ogInfo.SiteName
	}

	if data.Locale == "" {
		data.Locale = ogInfo.Locale
	}

	if ogInfo.Article != nil {
		data.PublishDate = ogInfo.Article.PublishedTime.Unix()

		if len(data.Tags) == 0 {
			data.Tags = ogInfo.Article.Tags
		}
	}
}
