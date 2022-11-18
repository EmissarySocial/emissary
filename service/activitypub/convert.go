package activitypub

import (
	"github.com/EmissarySocial/emissary/model"
	"github.com/benpate/data"
	"github.com/go-fed/activity/streams/vocab"
)

func ToActivityStream(item data.Object, itemType string) (vocab.Type, error) {

}

func ToInboxItem(item vocab.Type) (model.InboxItem, error) {

	result := model.NewInboxItem()

	return result, nil
}

func ToOutboxItem(item vocab.Type) (model.OutboxItem, error) {

	result := model.NewOutboxItem()

	return result, nil
}
