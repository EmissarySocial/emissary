package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
)

func StreamToGorillaFeed(stream model.Stream) *feeds.Item {
	return &feeds.Item{
		Title:       stream.Document.Label,
		Description: stream.Document.Summary,
		Content:     stream.Content.HTML,
		Link: &feeds.Link{
			Href: stream.Document.URL,
		},
		Author: &feeds.Author{
			Name:  stream.Document.Author.Name,
			Email: stream.Document.Author.EmailAddress,
		},
		Created: time.UnixMilli(stream.PublishDate),
	}
}
