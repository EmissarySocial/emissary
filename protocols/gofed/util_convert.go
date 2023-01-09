package gofed

import (
	"time"

	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/derp"
	"github.com/benpate/rosetta/mapof"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

func ToGoFed(item model.Activity) (vocab.Type, error) {
	manager := streams.Manager{}
	aliasMap := mapof.String{}

	jsonLD := mapof.Any{
		"type":    item.Document.Type,
		"id":      item.Document.URL,
		"name":    item.Document.Label,
		"summary": item.Document.Summary,
		"image":   item.Document.ImageURL,
		"author": mapof.Any{
			"id":    item.Document.Author.ProfileURL,
			"name":  item.Document.Author.Name,
			"image": item.Document.Author.ImageURL,
			"email": item.Document.Author.EmailAddress,
		},
		"published": time.UnixMilli(item.Document.PublishDate).Format(time.RFC3339),
	}

	switch jsonLD["type"] {

	case model.DocumentTypeArticle:
		return manager.DeserializeArticleActivityStreams()(jsonLD, aliasMap)

	case model.DocumentTypeNote:
		return manager.DeserializeNoteActivityStreams()(jsonLD, aliasMap)

	default:
		return nil, derp.NewInternalError("gofed.ToGoFed", "Unable to convert item to GoFed", item)

	}
}

func ToModel(item vocab.Type) (model.Activity, error) {

	result := model.NewActivity()
	data, err := streams.Serialize(item)

	if err != nil {
		return result, derp.Wrap(err, "gofed.ToModel", "Unable to serialize item", item)
	}

	spew.Dump("Debugging ToModel...", data)

	// TODO: CRITICAL: Map from dataset to result

	return result, nil
}
