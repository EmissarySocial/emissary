package convert

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/gorilla/feeds"
)

// StreamToGorillaFeed converts a Stream into an RSS/Atom feed item
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

	// `AUTHOR` PROPERTY REMOVED SO WE DON'T LEAK EMAIL ADDRESSES

	return result
}

// SearchResultToGorillaFeed converts a SearchResult into an RSS/Atom feed item
func SearchResultToGorillaFeed(searchResult model.SearchResult) *feeds.Item {
	result := &feeds.Item{
		Title:       searchResult.Name,
		Description: searchResult.Summary,
		Link: &feeds.Link{
			Href: searchResult.URL,
		},
	}

	if searchResult.CreateDate != 0 {
		result.Created = time.UnixMilli(searchResult.CreateDate)
	}

	if searchResult.AttributedTo != "" {
		result.Author = &feeds.Author{
			Name: searchResult.AttributedTo,
		}
	}

	return result
}
