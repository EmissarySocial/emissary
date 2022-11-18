package common

import (
	"github.com/EmissarySocial/emissary/gofed/activityStreams"
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/benpate/derp"
	"github.com/go-fed/activity/streams/vocab"
)

func ToActivityStream(item data.Object, documentType string) (vocab.Type, error) {

	result, err := activityStreams.NewDocument(documentType)

	if err != nil {
		return result, derp.Wrap(err, "activitypub.ToActivityStream", "Error creating ActivityStream document", documentType)
	}

	return result, nil
}

func ToModelObject(item vocab.Type) (model.InboxItem, error) {

	result := model.NewInboxItem()

	return result, nil
}
