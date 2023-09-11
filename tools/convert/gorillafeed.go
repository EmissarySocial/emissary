package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
)

func StreamToGorillaFeed(stream model.Stream) *feeds.Item {
	result := &feeds.Item{
		Title:       stream.Label,
		Description: stream.Summary,
		Content:     stream.Content.HTML,
		Link: &feeds.Link{
			Href: stream.URL,
		},
		Created: time.Unix(stream.PublishDate, 0),
	}

	if stream.AttributedTo.NotEmpty() {
		result.Author = &feeds.Author{
			Name:  stream.AttributedTo.Name,
			Email: stream.AttributedTo.EmailAddress,
		}
	}

	return result
}
