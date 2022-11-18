package activitypub

import (
	"github.com/EmissarySocial/emissary/model"
	as "github.com/benpate/activitystream"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

func ToActivityStream(item data.Object, documentType string) (vocab.Type, error) {

	result, err := activityStreamDocument(documentType)

	if err != nil {
		return result, derp.Wrap(err, "activitypub.ToActivityStream", "Error creating ActivityStream document", documentType)
	}
}

func ToInboxItem(item vocab.Type) (model.InboxItem, error) {

	result := model.NewInboxItem()

	return result, nil
}

func ToOutboxItem(item vocab.Type) (model.OutboxItem, error) {

	result := model.NewOutboxItem()

	return result, nil
}

func activityStreamDocument(documentType string) (vocab.Type, error) {

	switch documentType {

	case as.ObjectTypeArticle:
		return streams.NewActivityStreamsArticle(), nil

	case as.ObjectTypeAudio:
		return streams.NewActivityStreamsAudio(), nil

	case as.ObjectTypeDocument:
		return streams.NewActivityStreamsDocument(), nil

	case as.ObjectTypeEvent:
		return streams.NewActivityStreamsEvent(), nil

	case as.ObjectTypeImage:
		return streams.NewActivityStreamsImage(), nil

	case as.ObjectTypeNote:
		return streams.NewActivityStreamsNote(), nil

	case as.ObjectTypePage:
		return streams.NewActivityStreamsPage(), nil

	case as.ObjectTypePlace:
		return streams.NewActivityStreamsPlace(), nil

	case as.ObjectTypeProfile:
		return streams.NewActivityStreamsProfile(), nil

	case as.ObjectTypeRelationship:
		return streams.NewActivityStreamsRelationship(), nil

	case as.ObjectTypeTombstone:
		return streams.NewActivityStreamsTombstone(), nil

	case as.ObjectTypeVideo:
		return streams.NewActivityStreamsVideo(), nil

	}

	return nil, derp.NewInternalError("activitypub.ToActivityStream", "Unknown document type", documentType)
}
