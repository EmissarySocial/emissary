// ActivityStreams wraps the go-fed ActivityStreams library, providing a (hopefully)
// simpler interface to the underlying library.
package activityStreams

import (
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams"
	"github.com/go-fed/activity/streams/vocab"
)

// NewDocument creates a new ActivityStreams document of the specified type.
func NewDocument(documentType string) (vocab.Type, error) {

	switch documentType {

	case ObjectTypeArticle:
		return streams.NewActivityStreamsArticle(), nil

	case ObjectTypeAudio:
		return streams.NewActivityStreamsAudio(), nil

	case ObjectTypeDocument:
		return streams.NewActivityStreamsDocument(), nil

	case ObjectTypeEvent:
		return streams.NewActivityStreamsEvent(), nil

	case ObjectTypeImage:
		return streams.NewActivityStreamsImage(), nil

	case ObjectTypeNote:
		return streams.NewActivityStreamsNote(), nil

	case ObjectTypePage:
		return streams.NewActivityStreamsPage(), nil

	case ObjectTypePlace:
		return streams.NewActivityStreamsPlace(), nil

	case ObjectTypeProfile:
		return streams.NewActivityStreamsProfile(), nil

	case ObjectTypeRelationship:
		return streams.NewActivityStreamsRelationship(), nil

	case ObjectTypeTombstone:
		return streams.NewActivityStreamsTombstone(), nil

	case ObjectTypeVideo:
		return streams.NewActivityStreamsVideo(), nil

	}

	return nil, derp.NewInternalError("activitypub.ToActivityStream", "Unknown document type", documentType)
}
