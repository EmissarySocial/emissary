package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
)

func StreamToGorillaFeed(stream model.Stream) *feeds.Item {
	result := &feeds.Item{
		Title:       stream.Document.Label,
		Description: stream.Document.Summary,
		Content:     stream.Content.HTML,
		Link: &feeds.Link{
			Href: stream.Document.URL,
		},
		Created: time.UnixMilli(stream.PublishDate),
	}

	if !stream.Document.AttributedTo.IsEmpty() {
		author := stream.Document.AttributedTo.First()
		result.Author = &feeds.Author{
			Name:  author.Name,
			Email: author.EmailAddress,
		}
	}

	return result
}
