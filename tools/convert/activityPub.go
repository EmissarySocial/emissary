package convert

import (
	"context"
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/rosetta/mapof"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

func StreamToActivityPub(stream *model.Stream) (vocab.Type, error) {
	jsonLD := mapof.Any{
		"type":    stream.Document.Type,
		"id":      stream.Document.URL,
		"name":    stream.Document.Label,
		"summary": stream.Document.Summary,
		"image":   stream.Document.ImageURL,
		"author": mapof.Any{
			"id":    stream.Document.Author.ProfileURL,
			"name":  stream.Document.Author.Name,
			"image": stream.Document.Author.ImageURL,
			"email": stream.Document.Author.EmailAddress,
		},
		"published": time.UnixMilli(stream.Document.PublishDate).Format(time.RFC3339),
	}

	return streams.ToType(context.TODO(), jsonLD)
}
