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

	if !stream.AttributedTo.IsEmpty() {
		author := stream.AttributedTo.First()
		result.Author = &feeds.Author{
			Name:  author.Name,
			Email: author.EmailAddress,
		}
	}

	return result
}
